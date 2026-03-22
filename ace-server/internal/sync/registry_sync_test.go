package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicroldan/ans/shared/ace"
)

func TestBuildSyncRequest_NoVariants(t *testing.T) {
	p := ace.Product{
		ID:          "prod-1",
		Name:        "Widget",
		Description: "A fine widget",
		Price:       ace.Money{Amount: 1999, Currency: "USD"},
	}

	req := BuildSyncRequest(p, "gadgets", []string{"sale"}, ace.Location{Country: "AR", Region: "CABA"})

	if req.ProductID != "prod-1" {
		t.Errorf("ProductID = %q, want %q", req.ProductID, "prod-1")
	}
	if req.PriceRange.Min != 1999 || req.PriceRange.Max != 1999 {
		t.Errorf("PriceRange = {%d, %d}, want {1999, 1999}", req.PriceRange.Min, req.PriceRange.Max)
	}
	if req.PriceRange.Currency != "USD" {
		t.Errorf("Currency = %q, want %q", req.PriceRange.Currency, "USD")
	}
	if !req.InStock {
		t.Error("InStock = false, want true (no variants defaults to in stock)")
	}
	if req.Category != "gadgets" {
		t.Errorf("Category = %q, want %q", req.Category, "gadgets")
	}
	if len(req.VariantsSummary) != 0 {
		t.Errorf("VariantsSummary length = %d, want 0", len(req.VariantsSummary))
	}
}

func TestBuildSyncRequest_WithVariants_InStock(t *testing.T) {
	p := ace.Product{
		ID:          "prod-2",
		Name:        "T-Shirt",
		Description: "Cotton tee",
		Price:       ace.Money{Amount: 2500, Currency: "USD"},
		Variants: []ace.Variant{
			{ID: "v1", Name: "Small", Price: ace.Money{Amount: 2000, Currency: "USD"}, Inventory: 0},
			{ID: "v2", Name: "Medium", Price: ace.Money{Amount: 2500, Currency: "USD"}, Inventory: 5},
			{ID: "v3", Name: "Large", Price: ace.Money{Amount: 3000, Currency: "USD"}, Inventory: 3},
		},
	}

	req := BuildSyncRequest(p, "apparel", nil, ace.Location{Country: "US", Region: "CA"})

	if req.PriceRange.Min != 2000 {
		t.Errorf("PriceRange.Min = %d, want 2000", req.PriceRange.Min)
	}
	if req.PriceRange.Max != 3000 {
		t.Errorf("PriceRange.Max = %d, want 3000", req.PriceRange.Max)
	}
	if !req.InStock {
		t.Error("InStock = false, want true (Medium and Large have inventory)")
	}
	if len(req.VariantsSummary) != 3 {
		t.Fatalf("VariantsSummary length = %d, want 3", len(req.VariantsSummary))
	}
	expected := []string{"Small", "Medium", "Large"}
	for i, name := range expected {
		if req.VariantsSummary[i] != name {
			t.Errorf("VariantsSummary[%d] = %q, want %q", i, req.VariantsSummary[i], name)
		}
	}
}

func TestBuildSyncRequest_AllOutOfStock(t *testing.T) {
	p := ace.Product{
		ID:          "prod-3",
		Name:        "Rare Item",
		Description: "Sold out everywhere",
		Price:       ace.Money{Amount: 5000, Currency: "ARS"},
		Variants: []ace.Variant{
			{ID: "v1", Name: "Red", Price: ace.Money{Amount: 4500, Currency: "ARS"}, Inventory: 0},
			{ID: "v2", Name: "Blue", Price: ace.Money{Amount: 5500, Currency: "ARS"}, Inventory: 0},
		},
	}

	req := BuildSyncRequest(p, "collectibles", nil, ace.Location{Country: "AR", Region: "BA"})

	if req.InStock {
		t.Error("InStock = true, want false (all variants have zero inventory)")
	}
	if req.PriceRange.Min != 4500 {
		t.Errorf("PriceRange.Min = %d, want 4500", req.PriceRange.Min)
	}
	if req.PriceRange.Max != 5500 {
		t.Errorf("PriceRange.Max = %d, want 5500", req.PriceRange.Max)
	}
}

func TestClient_PushProduct(t *testing.T) {
	var receivedReq ace.ProductSyncRequest
	var receivedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ace.SyncResponse{Indexed: 1})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	syncReq := ace.ProductSyncRequest{
		ProductID: "prod-1",
		Name:      "Widget",
	}

	resp, err := client.PushProduct(syncReq)
	if err != nil {
		t.Fatalf("PushProduct: %v", err)
	}
	if resp.Indexed != 1 {
		t.Errorf("Indexed = %d, want 1", resp.Indexed)
	}
	if receivedAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer test-token")
	}
	if receivedReq.ProductID != "prod-1" {
		t.Errorf("received ProductID = %q, want %q", receivedReq.ProductID, "prod-1")
	}
}

func TestClient_DeleteProduct(t *testing.T) {
	var receivedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		if r.Method != http.MethodDelete {
			t.Errorf("Method = %q, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	if err := client.DeleteProduct("prod-99"); err != nil {
		t.Fatalf("DeleteProduct: %v", err)
	}
	if receivedPath != "/api/v1/sync/products/prod-99" {
		t.Errorf("path = %q, want %q", receivedPath, "/api/v1/sync/products/prod-99")
	}
}

func TestClient_PushProduct_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	_, err := client.PushProduct(ace.ProductSyncRequest{ProductID: "prod-1"})
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}
