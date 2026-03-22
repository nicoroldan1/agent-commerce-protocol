package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nicroldan/ans/shared/ace"
)

// ProductIndexer abstracts the search engine's indexing operations.
type ProductIndexer interface {
	IndexProduct(ctx context.Context, storeID, storeName string, p ace.ProductSyncRequest) error
	DeleteProduct(ctx context.Context, storeID, productID string) error
}

// TokenResolver abstracts token-to-store resolution.
type TokenResolver interface {
	ResolveToken(rawToken string) (storeID, storeName string, ok bool)
}

// SyncHandler handles product sync HTTP endpoints.
type SyncHandler struct {
	indexer  ProductIndexer
	resolver TokenResolver
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(indexer ProductIndexer, resolver TokenResolver) *SyncHandler {
	return &SyncHandler{indexer: indexer, resolver: resolver}
}

// RegisterRoutes registers all sync routes on the given mux.
func (h *SyncHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /registry/v1/products/sync", h.SyncProducts)
	mux.HandleFunc("DELETE /registry/v1/products/sync/{product_id}", h.DeleteSyncedProduct)
}

// SyncProducts handles POST /registry/v1/products/sync.
// Accepts a single ProductSyncRequest or a ProductBatchSyncRequest.
func (h *SyncHandler) SyncProducts(w http.ResponseWriter, r *http.Request) {
	storeID, storeName, ok := h.authenticate(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing registry token")
		return
	}

	// Try to decode as batch first, fall back to single product.
	products, err := decodeSyncBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body")
		return
	}

	resp := ace.SyncResponse{}
	for _, p := range products {
		if p.ProductID == "" {
			resp.Errors = append(resp.Errors, ace.SyncError{
				ProductID: "",
				Error:     "product_id is required",
			})
			continue
		}
		if err := h.indexer.IndexProduct(r.Context(), storeID, storeName, p); err != nil {
			resp.Errors = append(resp.Errors, ace.SyncError{
				ProductID: p.ProductID,
				Error:     err.Error(),
			})
			continue
		}
		resp.Indexed++
	}

	writeJSON(w, http.StatusOK, resp)
}

// DeleteSyncedProduct handles DELETE /registry/v1/products/sync/{product_id}.
func (h *SyncHandler) DeleteSyncedProduct(w http.ResponseWriter, r *http.Request) {
	storeID, _, ok := h.authenticate(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing registry token")
		return
	}

	productID := r.PathValue("product_id")
	if productID == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "product_id is required")
		return
	}

	if err := h.indexer.DeleteProduct(r.Context(), storeID, productID); err != nil {
		writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// authenticate extracts the Bearer token and resolves it to a store.
func (h *SyncHandler) authenticate(r *http.Request) (storeID, storeName string, ok bool) {
	token := extractToken(r)
	if token == "" {
		return "", "", false
	}
	return h.resolver.ResolveToken(token)
}

// extractToken extracts the Bearer token from the Authorization header.
func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	return strings.TrimPrefix(auth, prefix)
}

// decodeSyncBody attempts to decode the request body as a batch request first,
// then falls back to a single product request.
func decodeSyncBody(r *http.Request) ([]ace.ProductSyncRequest, error) {
	defer r.Body.Close()

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}

	// Try batch format: {"products": [...]}
	var batch ace.ProductBatchSyncRequest
	if err := json.Unmarshal(raw, &batch); err == nil && len(batch.Products) > 0 {
		return batch.Products, nil
	}

	// Fall back to single product.
	var single ace.ProductSyncRequest
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil, err
	}
	return []ace.ProductSyncRequest{single}, nil
}
