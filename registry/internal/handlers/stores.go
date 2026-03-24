package handlers

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nicroldan/ans/shared/ace"

	"github.com/nicroldan/ans/registry/internal/auth"
	"github.com/nicroldan/ans/registry/internal/healthcheck"
	"github.com/nicroldan/ans/registry/internal/store"
)

// ProductDeleter abstracts the ability to delete all indexed products for a store.
type ProductDeleter interface {
	DeleteByStore(ctx context.Context, storeID string) error
}

// StoreHandler handles store-related HTTP endpoints.
type StoreHandler struct {
	store   *store.MemoryStore
	deleter ProductDeleter
}

// NewStoreHandler creates a new StoreHandler. The deleter may be nil if search is unavailable.
func NewStoreHandler(s *store.MemoryStore, deleter ProductDeleter) *StoreHandler {
	return &StoreHandler{store: s, deleter: deleter}
}

// RegisterRoutes registers all store routes on the given mux.
func (h *StoreHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /registry/v1/stores", h.CreateStore)
	mux.HandleFunc("GET /registry/v1/stores", h.ListStores)
	mux.HandleFunc("GET /registry/v1/stores/{id}", h.GetStore)
	mux.HandleFunc("GET /registry/v1/stores/{id}/health", h.CheckHealth)
	mux.HandleFunc("POST /registry/v1/stores/{id}/report", h.ReportStore)
	mux.HandleFunc("DELETE /registry/v1/stores/{id}", h.DeleteStore)
}

// CreateStore handles POST /registry/v1/stores.
func (h *StoreHandler) CreateStore(w http.ResponseWriter, r *http.Request) {
	var reg ace.StoreRegistration
	if err := decodeJSON(r, &reg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body")
		return
	}

	if reg.WellKnownURL == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "well_known_url is required")
		return
	}

	// Validate by fetching the well-known URL.
	wk, err := healthcheck.FetchWellKnown(reg.WellKnownURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_store", err.Error())
		return
	}

	now := time.Now().UTC()
	entry := ace.StoreEntry{
		WellKnownURL: reg.WellKnownURL,
		Name:         wk.Name,
		Categories:   reg.Categories,
		Country:      reg.Country,
		Currencies:   wk.Currencies,
		Capabilities: wk.Capabilities,
		HealthStatus: "healthy",
		LastChecked:  now,
		RegisteredAt: now,
	}

	created := h.store.Create(entry)

	token := auth.GenerateToken()
	hash := auth.HashToken(token)
	h.store.StoreTokenHash(created.ID, hash)

	writeJSON(w, http.StatusCreated, ace.StoreRegistrationResponse{
		StoreEntry:    created,
		RegistryToken: token,
	})
}

// ListStores handles GET /registry/v1/stores.
func (h *StoreHandler) ListStores(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	filters := store.ListFilters{
		Query:    q.Get("q"),
		Category: q.Get("category"),
		Country:  q.Get("country"),
		Currency: q.Get("currency"),
		Offset:   offset,
		Limit:    limit,
	}

	data, total := h.store.List(filters)

	writeJSON(w, http.StatusOK, ace.PaginatedResponse[ace.StoreEntry]{
		Data:   data,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	})
}

// GetStore handles GET /registry/v1/stores/{id}.
func (h *StoreHandler) GetStore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	entry, ok := h.store.GetByID(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Store not found")
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

// CheckHealth handles GET /registry/v1/stores/{id}/health.
func (h *StoreHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	entry, ok := h.store.GetByID(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Store not found")
		return
	}

	_, err := healthcheck.FetchWellKnown(entry.WellKnownURL)
	now := time.Now().UTC()
	entry.LastChecked = now

	if err != nil {
		entry.HealthStatus = "down"
	} else {
		entry.HealthStatus = "healthy"
	}

	h.store.Update(entry)
	writeJSON(w, http.StatusOK, entry)
}

// ReportStore handles POST /registry/v1/stores/{id}/report.
func (h *StoreHandler) ReportStore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, ok := h.store.GetByID(id); !ok {
		writeError(w, http.StatusNotFound, "not_found", "Store not found")
		return
	}

	var body struct {
		Reason  string `json:"reason"`
		Details string `json:"details"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body")
		return
	}

	if body.Reason == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "reason is required")
		return
	}

	report := ace.StoreReport{
		StoreID:   id,
		Reason:    body.Reason,
		Details:   body.Details,
		CreatedAt: time.Now().UTC(),
	}
	h.store.AddReport(report)

	log.Printf("Store %s reported: reason=%s details=%s", id, body.Reason, body.Details)

	writeJSON(w, http.StatusOK, map[string]string{"status": "reported"})
}

// DeleteStore handles DELETE /registry/v1/stores/{id}.
func (h *StoreHandler) DeleteStore(w http.ResponseWriter, r *http.Request) {
	// Require admin token.
	adminToken := os.Getenv("REGISTRY_ADMIN_TOKEN")
	if adminToken == "" {
		writeError(w, http.StatusInternalServerError, "server_error", "Admin token not configured")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != adminToken {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing admin token")
		return
	}

	id := r.PathValue("id")

	if !h.store.DeleteStore(id) {
		writeError(w, http.StatusNotFound, "not_found", "Store not found")
		return
	}

	// Remove indexed products if search engine is available.
	if h.deleter != nil {
		if err := h.deleter.DeleteByStore(r.Context(), id); err != nil {
			log.Printf("WARNING: Failed to delete indexed products for store %s: %v", id, err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
