package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/nicroldan/ans/shared/ace"
	"github.com/nicroldan/ans/ace-server/internal/audit"
	"github.com/nicroldan/ans/ace-server/internal/middleware"
	"github.com/nicroldan/ans/ace-server/internal/store"
)

// BuyerHandler implements the ACE Buyer API.
type BuyerHandler struct {
	store   *store.MemoryStore
	audit   *audit.Logger
	storeID string
	name    string
	baseURL string
}

// NewBuyerHandler creates a new BuyerHandler.
func NewBuyerHandler(s *store.MemoryStore, al *audit.Logger, storeID, name, baseURL string) *BuyerHandler {
	return &BuyerHandler{
		store:   s,
		audit:   al,
		storeID: storeID,
		name:    name,
		baseURL: baseURL,
	}
}

// Discovery handles GET /.well-known/agent-commerce
func (h *BuyerHandler) Discovery(w http.ResponseWriter, r *http.Request) {
	resp := ace.WellKnownResponse{
		StoreID:    h.storeID,
		Name:       h.name,
		Version:    "1.0.0",
		ACEBaseURL: h.baseURL + "/ace/v1",
		Capabilities: []string{
			"catalog", "cart", "orders", "payments", "shipping",
		},
		Auth: ace.AuthConfig{
			Type:   "api_key",
			Header: "X-ACE-Key",
		},
		Currencies: []string{"USD"},
	}
	writeJSON(w, http.StatusOK, resp)
}

// ListProducts handles GET /ace/v1/products
func (h *BuyerHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := q.Get("q")
	category := q.Get("category")
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	products, total := h.store.ListProducts("published", category, query, offset, limit)

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "catalog.list", actor, actorType, "products", "", nil)

	writeJSON(w, http.StatusOK, ace.PaginatedResponse[ace.Product]{
		Data:   products,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	})
}

// GetProduct handles GET /ace/v1/products/{id}
func (h *BuyerHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, ok := h.store.GetProduct(id)
	if !ok || p.Status != "published" {
		writeError(w, http.StatusNotFound, "not_found", "Product not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ShippingQuote handles POST /ace/v1/shipping/quote
func (h *BuyerHandler) ShippingQuote(w http.ResponseWriter, r *http.Request) {
	var req ace.ShippingQuoteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	quote := ace.ShippingQuote{
		Options: []ace.ShippingOption{
			{
				ID:            "ship_standard",
				Name:          "Standard Shipping",
				Price:         ace.Money{Amount: 599, Currency: "USD"},
				EstimatedDays: 7,
			},
			{
				ID:            "ship_express",
				Name:          "Express Shipping",
				Price:         ace.Money{Amount: 1299, Currency: "USD"},
				EstimatedDays: 3,
			},
			{
				ID:            "ship_overnight",
				Name:          "Overnight Shipping",
				Price:         ace.Money{Amount: 2499, Currency: "USD"},
				EstimatedDays: 1,
			},
		},
	}
	writeJSON(w, http.StatusOK, quote)
}

// CreateCart handles POST /ace/v1/cart
func (h *BuyerHandler) CreateCart(w http.ResponseWriter, r *http.Request) {
	cart := h.store.CreateCart()

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "cart.create", actor, actorType, "cart", cart.ID, nil)

	writeJSON(w, http.StatusCreated, cart)
}

// AddCartItem handles POST /ace/v1/cart/{id}/items
func (h *BuyerHandler) AddCartItem(w http.ResponseWriter, r *http.Request) {
	cartID := r.PathValue("id")

	var req ace.AddCartItemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_quantity", "Quantity must be positive")
		return
	}

	// Validate product exists and is published
	product, ok := h.store.GetProduct(req.ProductID)
	if !ok || product.Status != "published" {
		writeError(w, http.StatusNotFound, "product_not_found", "Product not found or not available")
		return
	}

	// Determine price
	price := product.Price
	if req.VariantID != "" {
		found := false
		for _, v := range product.Variants {
			if v.ID == req.VariantID {
				price = v.Price
				if v.Inventory < req.Quantity {
					writeError(w, http.StatusConflict, "insufficient_stock", "Not enough inventory")
					return
				}
				found = true
				break
			}
		}
		if !found {
			writeError(w, http.StatusNotFound, "variant_not_found", "Variant not found")
			return
		}
	} else if len(product.Variants) > 0 {
		// Use first variant price and check stock
		v := product.Variants[0]
		price = v.Price
		if v.Inventory < req.Quantity {
			writeError(w, http.StatusConflict, "insufficient_stock", "Not enough inventory")
			return
		}
	}

	item := ace.CartItem{
		ProductID: req.ProductID,
		VariantID: req.VariantID,
		Quantity:  req.Quantity,
		Price:     price,
	}

	cart, ok := h.store.AddCartItem(cartID, item)
	if !ok {
		writeError(w, http.StatusNotFound, "cart_not_found", "Cart not found")
		return
	}

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "cart.add_item", actor, actorType, "cart", cartID, map[string]any{
		"product_id": req.ProductID,
		"quantity":   req.Quantity,
	})

	writeJSON(w, http.StatusOK, cart)
}

// GetCart handles GET /ace/v1/cart/{id}
func (h *BuyerHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cart, ok := h.store.GetCart(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Cart not found")
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

// CreateOrder handles POST /ace/v1/orders
func (h *BuyerHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req ace.CreateOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	cart, ok := h.store.GetCart(req.CartID)
	if !ok {
		writeError(w, http.StatusNotFound, "cart_not_found", "Cart not found")
		return
	}
	if len(cart.Items) == 0 {
		writeError(w, http.StatusBadRequest, "empty_cart", "Cart has no items")
		return
	}

	// Build order items and decrement inventory
	orderItems := make([]ace.OrderItem, 0, len(cart.Items))
	for _, ci := range cart.Items {
		product, exists := h.store.GetProduct(ci.ProductID)
		if !exists {
			writeError(w, http.StatusConflict, "product_unavailable", fmt.Sprintf("Product %s no longer available", ci.ProductID))
			return
		}
		if !h.store.DecrementInventory(ci.ProductID, ci.VariantID, ci.Quantity) {
			writeError(w, http.StatusConflict, "insufficient_stock", fmt.Sprintf("Insufficient stock for %s", product.Name))
			return
		}
		orderItems = append(orderItems, ace.OrderItem{
			ProductID:   ci.ProductID,
			ProductName: product.Name,
			VariantID:   ci.VariantID,
			Quantity:    ci.Quantity,
			Price:       ci.Price,
		})
	}

	order := &ace.Order{
		CartID: cart.ID,
		Items:  orderItems,
		Total:  cart.Total,
		Status: "pending",
	}
	h.store.CreateOrder(order)

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "order.create", actor, actorType, "order", order.ID, map[string]any{
		"cart_id": cart.ID,
		"total":   order.Total.Amount,
	})

	writeJSON(w, http.StatusCreated, order)
}

// GetOrder handles GET /ace/v1/orders/{id}
func (h *BuyerHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	order, ok := h.store.GetOrder(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Order not found")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

// Pay handles POST /ace/v1/orders/{id}/pay
func (h *BuyerHandler) Pay(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")

	order, ok := h.store.GetOrder(orderID)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Order not found")
		return
	}

	if order.Status != "pending" {
		writeError(w, http.StatusConflict, "invalid_status", "Order is not in pending status")
		return
	}

	var req ace.InitiatePaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid request body")
		return
	}

	if req.Provider == "" {
		req.Provider = "mock"
	}

	payment := &ace.Payment{
		OrderID:    orderID,
		Status:     "processing",
		Provider:   req.Provider,
		Amount:     order.Total,
		ExternalID: fmt.Sprintf("mock_ext_%s", orderID),
		PaymentURL: fmt.Sprintf("https://pay.example.com/mock/%s", orderID),
	}
	h.store.CreatePayment(payment)
	h.store.SetOrderPayment(orderID, payment)

	// Simulate auto-complete payment
	h.store.SetPaymentStatus(payment.ID, "completed")
	payment.Status = "completed"
	h.store.SetOrderStatus(orderID, "paid")

	actor, actorType := middleware.GetActor(r)
	h.audit.Log(r.Context(), h.storeID, "payment.create", actor, actorType, "payment", payment.ID, map[string]any{
		"order_id": orderID,
		"provider": req.Provider,
		"amount":   payment.Amount.Amount,
	})

	writeJSON(w, http.StatusCreated, payment)
}

// PaymentStatus handles GET /ace/v1/orders/{id}/pay/status
func (h *BuyerHandler) PaymentStatus(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	payment, ok := h.store.GetPaymentByOrderID(orderID)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Payment not found for this order")
		return
	}
	writeJSON(w, http.StatusOK, payment)
}
