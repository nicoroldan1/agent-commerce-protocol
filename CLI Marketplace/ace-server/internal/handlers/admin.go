package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/nicroldan/ans/shared/ace"
	"github.com/nicroldan/ans/ace-server/internal/audit"
	"github.com/nicroldan/ans/ace-server/internal/middleware"
	"github.com/nicroldan/ans/ace-server/internal/policy"
	"github.com/nicroldan/ans/ace-server/internal/store"
)

// AdminHandler implements the Seller Admin API.
type AdminHandler struct {
	store   *store.MemoryStore
	audit   *audit.Logger
	policy  *policy.Engine
	storeID string
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(s *store.MemoryStore, al *audit.Logger, pe *policy.Engine, storeID string) *AdminHandler {
	return &AdminHandler{
		store:   s,
		audit:   al,
		policy:  pe,
		storeID: storeID,
	}
}

// --- Catalog Management ---

// CreateProduct handles POST /api/v1/stores/{store_id}/products
func (h *AdminHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p ace.Product
	if err := decodeJSON(r, &p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}
	p.Status = "draft"
	h.store.CreateProduct(&p)

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "product.create", actor, actorType, "product", p.ID, nil)

	writeJSON(w, http.StatusCreated, p)
}

// ListProducts handles GET /api/v1/stores/{store_id}/products
func (h *AdminHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	category := q.Get("category")
	query := q.Get("q")
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	products, total := h.store.ListProducts(status, category, query, offset, limit)

	writeJSON(w, http.StatusOK, ace.PaginatedResponse[ace.Product]{
		Data:   products,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	})
}

// UpdateProduct handles PATCH /api/v1/stores/{store_id}/products/{id}
func (h *AdminHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var updates map[string]any
	if err := decodeJSON(r, &updates); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	// Parse price if present
	parsed := make(map[string]any)
	for k, v := range updates {
		if k == "price" {
			if priceMap, ok := v.(map[string]any); ok {
				amount, _ := priceMap["amount"].(float64)
				currency, _ := priceMap["currency"].(string)
				parsed["price"] = ace.Money{Amount: int64(amount), Currency: currency}
			}
		} else if k == "variants" {
			// Re-marshal and unmarshal variants
			b, _ := json.Marshal(v)
			var variants []ace.Variant
			json.Unmarshal(b, &variants)
			parsed["variants"] = variants
		} else {
			parsed[k] = v
		}
	}

	p, ok := h.store.UpdateProduct(id, parsed)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Product not found")
		return
	}

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "product.update", actor, actorType, "product", id, nil)

	writeJSON(w, http.StatusOK, p)
}

// DeleteProduct handles DELETE /api/v1/stores/{store_id}/products/{id}
func (h *AdminHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !h.store.DeleteProduct(id) {
		writeError(w, http.StatusNotFound, "not_found", "Product not found")
		return
	}

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "product.delete", actor, actorType, "product", id, nil)

	w.WriteHeader(http.StatusNoContent)
}

// PublishProduct handles POST /api/v1/stores/{store_id}/products/{id}/publish
func (h *AdminHandler) PublishProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	actor, actorType := middleware.GetActor(r)

	// Check policy
	result := h.policy.Check("product.publish", actor, actorType, id)
	switch result.Effect {
	case "deny":
		writeError(w, http.StatusForbidden, "policy_denied", "Action denied by policy")
		return
	case "needs_approval":
		h.audit.Log(r.Context(), h.storeID, "product.publish.approval_requested", actor, actorType, "product", id, nil)
		writeJSON(w, http.StatusAccepted, result.Approval)
		return
	}

	p, ok := h.store.SetProductStatus(id, "published")
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Product not found")
		return
	}

	h.audit.Log(r.Context(), h.storeID, "product.publish", actor, actorType, "product", id, nil)

	writeJSON(w, http.StatusOK, p)
}

// UnpublishProduct handles POST /api/v1/stores/{store_id}/products/{id}/unpublish
func (h *AdminHandler) UnpublishProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	p, ok := h.store.SetProductStatus(id, "unpublished")
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Product not found")
		return
	}

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "product.unpublish", actor, actorType, "product", id, nil)

	writeJSON(w, http.StatusOK, p)
}

// --- Inventory ---

// UpdateVariantInventory handles PATCH /api/v1/stores/{store_id}/variants/{id}/inventory
func (h *AdminHandler) UpdateVariantInventory(w http.ResponseWriter, r *http.Request) {
	variantID := r.PathValue("id")

	var body struct {
		Inventory int `json:"inventory"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	if !h.store.UpdateVariantInventory(variantID, body.Inventory) {
		writeError(w, http.StatusNotFound, "not_found", "Variant not found")
		return
	}

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "inventory.update", actor, actorType, "variant", variantID, map[string]any{
		"inventory": body.Inventory,
	})

	writeJSON(w, http.StatusOK, map[string]any{"variant_id": variantID, "inventory": body.Inventory})
}

// --- Orders ---

// ListOrders handles GET /api/v1/stores/{store_id}/orders
func (h *AdminHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	orders, total := h.store.ListOrders(offset, limit)

	writeJSON(w, http.StatusOK, ace.PaginatedResponse[ace.Order]{
		Data:   orders,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	})
}

// GetOrder handles GET /api/v1/stores/{store_id}/orders/{id}
func (h *AdminHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	order, ok := h.store.GetOrder(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Order not found")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

// FulfillOrder handles POST /api/v1/stores/{store_id}/orders/{id}/fulfill
func (h *AdminHandler) FulfillOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	order, ok := h.store.GetOrder(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Order not found")
		return
	}
	if order.Status != "paid" {
		writeError(w, http.StatusConflict, "invalid_status", "Order must be paid before fulfillment")
		return
	}

	order, _ = h.store.SetOrderStatus(id, "fulfilled")

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "order.fulfill", actor, actorType, "order", id, nil)

	writeJSON(w, http.StatusOK, order)
}

// RefundOrder handles POST /api/v1/stores/{store_id}/orders/{id}/refund
func (h *AdminHandler) RefundOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	actor, actorType := middleware.GetActor(r)

	// Check policy
	result := h.policy.Check("order.refund", actor, actorType, id)
	switch result.Effect {
	case "deny":
		writeError(w, http.StatusForbidden, "policy_denied", "Action denied by policy")
		return
	case "needs_approval":
		h.audit.Log(r.Context(), h.storeID, "order.refund.approval_requested", actor, actorType, "order", id, nil)
		writeJSON(w, http.StatusAccepted, result.Approval)
		return
	}

	order, ok := h.store.SetOrderStatus(id, "refunded")
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Order not found")
		return
	}

	h.audit.Log(r.Context(), h.storeID, "order.refund", actor, actorType, "order", id, nil)

	writeJSON(w, http.StatusOK, order)
}

// --- Policies ---

// GetPolicies handles GET /api/v1/stores/{store_id}/policies
func (h *AdminHandler) GetPolicies(w http.ResponseWriter, r *http.Request) {
	policies := h.store.GetPolicies()
	writeJSON(w, http.StatusOK, policies)
}

// UpdatePolicies handles PUT /api/v1/stores/{store_id}/policies
func (h *AdminHandler) UpdatePolicies(w http.ResponseWriter, r *http.Request) {
	var policies []ace.Policy
	if err := decodeJSON(r, &policies); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}
	h.store.SetPolicies(policies)

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "policies.update", actor, actorType, "policies", "", nil)

	writeJSON(w, http.StatusOK, h.store.GetPolicies())
}

// --- Approvals ---

// ListApprovals handles GET /api/v1/stores/{store_id}/approvals
func (h *AdminHandler) ListApprovals(w http.ResponseWriter, r *http.Request) {
	approvals := h.store.ListPendingApprovals()
	if approvals == nil {
		approvals = []ace.Approval{}
	}
	writeJSON(w, http.StatusOK, approvals)
}

// ApproveApproval handles POST /api/v1/stores/{store_id}/approvals/{id}/approve
func (h *AdminHandler) ApproveApproval(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	actor, actorType := middleware.GetActor(r)

	approval, ok := h.store.ResolveApproval(id, "approved", actor)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Approval not found")
		return
	}

	h.audit.Log(r.Context(), h.storeID, "approval.approve", actor, actorType, "approval", id, nil)

	writeJSON(w, http.StatusOK, approval)
}

// RejectApproval handles POST /api/v1/stores/{store_id}/approvals/{id}/reject
func (h *AdminHandler) RejectApproval(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	actor, actorType := middleware.GetActor(r)

	approval, ok := h.store.ResolveApproval(id, "rejected", actor)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Approval not found")
		return
	}

	h.audit.Log(r.Context(), h.storeID, "approval.reject", actor, actorType, "approval", id, nil)

	writeJSON(w, http.StatusOK, approval)
}

// --- Audit ---

// ListAuditLogs handles GET /api/v1/stores/{store_id}/audit-logs
func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	action := q.Get("action")
	actor := q.Get("actor")
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	storeID := r.PathValue("store_id")
	entries, total := h.store.QueryAuditLog(storeID, action, actor, offset, limit)

	writeJSON(w, http.StatusOK, ace.PaginatedResponse[ace.AuditEntry]{
		Data:   entries,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	})
}

// --- API Keys ---

// CreateAPIKey handles POST /api/v1/stores/{store_id}/api-keys
func (h *AdminHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req ace.CreateAPIKeyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "missing_name", "Name is required")
		return
	}

	resp, _ := h.store.CreateAPIKey(req.Name, req.Scopes)

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "apikey.create", actor, actorType, "apikey", resp.ID, map[string]any{
		"name": req.Name,
	})

	writeJSON(w, http.StatusCreated, resp)
}

// ListAPIKeys handles GET /api/v1/stores/{store_id}/api-keys
func (h *AdminHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys := h.store.ListAPIKeys()
	writeJSON(w, http.StatusOK, keys)
}

// DeleteAPIKey handles DELETE /api/v1/stores/{store_id}/api-keys/{id}
func (h *AdminHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !h.store.DeleteAPIKey(id) {
		writeError(w, http.StatusNotFound, "not_found", "API key not found")
		return
	}

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "apikey.delete", actor, actorType, "apikey", id, nil)

	w.WriteHeader(http.StatusNoContent)
}
