package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nicroldan/ans/shared/ace"
	"github.com/nicroldan/ans/ace-server/internal/audit"
	"github.com/nicroldan/ans/ace-server/internal/handlers"
	"github.com/nicroldan/ans/ace-server/internal/middleware"
	"github.com/nicroldan/ans/ace-server/internal/policy"
	"github.com/nicroldan/ans/ace-server/internal/store"
)

func main() {
	port := envOrDefault("PORT", "8081")
	storeID := envOrDefault("STORE_ID", "store_demo_001")
	storeName := envOrDefault("STORE_NAME", "ACE Demo Store")
	adminToken := envOrDefault("ADMIN_TOKEN", generateSecureToken())
	baseURL := envOrDefault("BASE_URL", fmt.Sprintf("http://localhost:%s", port))

	// Initialize stores and services
	memStore := store.New()
	auditLogger := audit.NewLogger(memStore)
	policyEngine := policy.NewEngine(memStore)

	// Set default policies
	memStore.SetPolicies(policy.DefaultPolicies())

	// Seed demo data
	demoKey := seedDemoData(memStore, storeID)

	// Create handlers
	buyerHandler := handlers.NewBuyerHandler(memStore, auditLogger, storeID, storeName, baseURL)
	adminHandler := handlers.NewAdminHandler(memStore, auditLogger, policyEngine, storeID)

	// Build router
	mux := http.NewServeMux()

	// Discovery (no auth)
	mux.HandleFunc("GET /.well-known/agent-commerce", buyerHandler.Discovery)

	// Buyer API routes (ACE auth)
	aceAuth := func(handler http.HandlerFunc) http.Handler {
		return middleware.ACEAuth(memStore, http.HandlerFunc(handler))
	}
	mux.Handle("GET /ace/v1/products", aceAuth(buyerHandler.ListProducts))
	mux.Handle("GET /ace/v1/products/{id}", aceAuth(buyerHandler.GetProduct))
	mux.Handle("POST /ace/v1/shipping/quote", aceAuth(buyerHandler.ShippingQuote))
	mux.Handle("POST /ace/v1/cart", aceAuth(buyerHandler.CreateCart))
	mux.Handle("POST /ace/v1/cart/{id}/items", aceAuth(buyerHandler.AddCartItem))
	mux.Handle("GET /ace/v1/cart/{id}", aceAuth(buyerHandler.GetCart))
	mux.Handle("POST /ace/v1/orders", aceAuth(buyerHandler.CreateOrder))
	mux.Handle("GET /ace/v1/orders/{id}", aceAuth(buyerHandler.GetOrder))
	mux.Handle("POST /ace/v1/orders/{id}/pay", aceAuth(buyerHandler.Pay))
	mux.Handle("GET /ace/v1/orders/{id}/pay/status", aceAuth(buyerHandler.PaymentStatus))

	// Admin API routes (admin auth)
	adminAuth := func(handler http.HandlerFunc) http.Handler {
		return middleware.AdminAuth(adminToken, http.HandlerFunc(handler))
	}
	mux.Handle("POST /api/v1/stores/{store_id}/products", adminAuth(adminHandler.CreateProduct))
	mux.Handle("GET /api/v1/stores/{store_id}/products", adminAuth(adminHandler.ListProducts))
	mux.Handle("PATCH /api/v1/stores/{store_id}/products/{id}", adminAuth(adminHandler.UpdateProduct))
	mux.Handle("DELETE /api/v1/stores/{store_id}/products/{id}", adminAuth(adminHandler.DeleteProduct))
	mux.Handle("POST /api/v1/stores/{store_id}/products/{id}/publish", adminAuth(adminHandler.PublishProduct))
	mux.Handle("POST /api/v1/stores/{store_id}/products/{id}/unpublish", adminAuth(adminHandler.UnpublishProduct))
	mux.Handle("PATCH /api/v1/stores/{store_id}/variants/{id}/inventory", adminAuth(adminHandler.UpdateVariantInventory))
	mux.Handle("GET /api/v1/stores/{store_id}/orders", adminAuth(adminHandler.ListOrders))
	mux.Handle("GET /api/v1/stores/{store_id}/orders/{id}", adminAuth(adminHandler.GetOrder))
	mux.Handle("POST /api/v1/stores/{store_id}/orders/{id}/fulfill", adminAuth(adminHandler.FulfillOrder))
	mux.Handle("POST /api/v1/stores/{store_id}/orders/{id}/refund", adminAuth(adminHandler.RefundOrder))
	mux.Handle("GET /api/v1/stores/{store_id}/policies", adminAuth(adminHandler.GetPolicies))
	mux.Handle("PUT /api/v1/stores/{store_id}/policies", adminAuth(adminHandler.UpdatePolicies))
	mux.Handle("GET /api/v1/stores/{store_id}/approvals", adminAuth(adminHandler.ListApprovals))
	mux.Handle("POST /api/v1/stores/{store_id}/approvals/{id}/approve", adminAuth(adminHandler.ApproveApproval))
	mux.Handle("POST /api/v1/stores/{store_id}/approvals/{id}/reject", adminAuth(adminHandler.RejectApproval))
	mux.Handle("GET /api/v1/stores/{store_id}/audit-logs", adminAuth(adminHandler.ListAuditLogs))
	mux.Handle("POST /api/v1/stores/{store_id}/api-keys", adminAuth(adminHandler.CreateAPIKey))
	mux.Handle("GET /api/v1/stores/{store_id}/api-keys", adminAuth(adminHandler.ListAPIKeys))
	mux.Handle("DELETE /api/v1/stores/{store_id}/api-keys/{id}", adminAuth(adminHandler.DeleteAPIKey))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("ACE Reference Server starting on :%s", port)
		log.Printf("Store: %s (%s)", storeName, storeID)
		log.Printf("Admin token: %s", adminToken)
		log.Printf("Demo API key: %s", demoKey)
		log.Printf("Discovery: %s/.well-known/agent-commerce", baseURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
	log.Println("Server stopped")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate secure token: %v", err)
	}
	return hex.EncodeToString(b)
}

func seedDemoData(s *store.MemoryStore, storeID string) string {
	products := []ace.Product{
		{
			Name:        "Wireless Headphones",
			Description: "Premium noise-cancelling wireless headphones with 30-hour battery life",
			Price:       ace.Money{Amount: 7999, Currency: "USD"},
			Status:      "published",
			Variants: []ace.Variant{
				{ID: "var_wh_black", Name: "Black", SKU: "WH-001-BLK", Price: ace.Money{Amount: 7999, Currency: "USD"}, Inventory: 150, Attributes: map[string]string{"color": "black", "category": "Electronics"}},
				{ID: "var_wh_white", Name: "White", SKU: "WH-001-WHT", Price: ace.Money{Amount: 7999, Currency: "USD"}, Inventory: 100, Attributes: map[string]string{"color": "white", "category": "Electronics"}},
			},
		},
		{
			Name:        "Organic Coffee Beans",
			Description: "Single-origin Ethiopian Yirgacheffe, medium roast, 1lb bag",
			Price:       ace.Money{Amount: 2499, Currency: "USD"},
			Status:      "published",
			Variants: []ace.Variant{
				{ID: "var_cof_med", Name: "Medium Roast", SKU: "COF-ETH-MED", Price: ace.Money{Amount: 2499, Currency: "USD"}, Inventory: 200, Attributes: map[string]string{"roast": "medium", "category": "Food & Beverage"}},
				{ID: "var_cof_dark", Name: "Dark Roast", SKU: "COF-ETH-DRK", Price: ace.Money{Amount: 2499, Currency: "USD"}, Inventory: 180, Attributes: map[string]string{"roast": "dark", "category": "Food & Beverage"}},
			},
		},
		{
			Name:        "Mechanical Keyboard",
			Description: "Cherry MX Brown switches, full RGB, hot-swappable, aluminum frame",
			Price:       ace.Money{Amount: 14999, Currency: "USD"},
			Status:      "published",
			Variants: []ace.Variant{
				{ID: "var_kb_brown", Name: "Cherry MX Brown", SKU: "KB-MX-BRN", Price: ace.Money{Amount: 14999, Currency: "USD"}, Inventory: 75, Attributes: map[string]string{"switch": "brown", "category": "Electronics"}},
				{ID: "var_kb_red", Name: "Cherry MX Red", SKU: "KB-MX-RED", Price: ace.Money{Amount: 14999, Currency: "USD"}, Inventory: 60, Attributes: map[string]string{"switch": "red", "category": "Electronics"}},
			},
		},
		{
			Name:        "Running Shoes",
			Description: "Lightweight performance running shoes with responsive cushioning",
			Price:       ace.Money{Amount: 12999, Currency: "USD"},
			Status:      "published",
			Variants: []ace.Variant{
				{ID: "var_shoe_9", Name: "Size 9", SKU: "SHOE-RUN-9", Price: ace.Money{Amount: 12999, Currency: "USD"}, Inventory: 50, Attributes: map[string]string{"size": "9", "category": "Sports"}},
				{ID: "var_shoe_10", Name: "Size 10", SKU: "SHOE-RUN-10", Price: ace.Money{Amount: 12999, Currency: "USD"}, Inventory: 65, Attributes: map[string]string{"size": "10", "category": "Sports"}},
			},
		},
		{
			Name:        "Python Programming Book",
			Description: "Comprehensive guide to Python programming, 4th edition, 800 pages",
			Price:       ace.Money{Amount: 3999, Currency: "USD"},
			Status:      "published",
			Variants: []ace.Variant{
				{ID: "var_book_paper", Name: "Paperback", SKU: "BOOK-PY-PBK", Price: ace.Money{Amount: 3999, Currency: "USD"}, Inventory: 120, Attributes: map[string]string{"format": "paperback", "category": "Books"}},
				{ID: "var_book_ebook", Name: "eBook", SKU: "BOOK-PY-EBK", Price: ace.Money{Amount: 2499, Currency: "USD"}, Inventory: 999, Attributes: map[string]string{"format": "ebook", "category": "Books"}},
			},
		},
		{
			Name:        "Yoga Mat",
			Description: "Non-slip premium yoga mat, 6mm thick, eco-friendly materials",
			Price:       ace.Money{Amount: 4999, Currency: "USD"},
			Status:      "published",
			Variants: []ace.Variant{
				{ID: "var_mat_purple", Name: "Purple", SKU: "MAT-YG-PRP", Price: ace.Money{Amount: 4999, Currency: "USD"}, Inventory: 90, Attributes: map[string]string{"color": "purple", "category": "Sports"}},
			},
		},
		{
			Name:        "Stainless Steel Water Bottle",
			Description: "Double-walled insulated bottle, keeps drinks cold 24h/hot 12h, 32oz",
			Price:       ace.Money{Amount: 3499, Currency: "USD"},
			Status:      "published",
			Variants: []ace.Variant{
				{ID: "var_bottle_blue", Name: "Ocean Blue", SKU: "BTL-SS-BLU", Price: ace.Money{Amount: 3499, Currency: "USD"}, Inventory: 175, Attributes: map[string]string{"color": "blue", "category": "Home & Kitchen"}},
			},
		},
	}

	for i := range products {
		s.CreateProduct(&products[i])
	}

	// Create demo API key (generated at startup, never hardcoded)
	demoKeyValue := envOrDefault("DEMO_API_KEY", "")
	var resp ace.CreateAPIKeyResponse
	var demoKey string
	if demoKeyValue != "" {
		resp = s.CreateAPIKeyWithValue("demo-agent", []string{"catalog:read", "cart:write", "orders:write", "payments:write"}, demoKeyValue)
		demoKey = demoKeyValue
	} else {
		resp, demoKey = s.CreateAPIKey("demo-agent", []string{"catalog:read", "cart:write", "orders:write", "payments:write"})
	}
	_ = resp

	_ = storeID
	return demoKey
}
