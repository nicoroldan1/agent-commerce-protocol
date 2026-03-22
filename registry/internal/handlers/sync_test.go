package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicroldan/ans/shared/ace"
)

// --- Mock implementations ---

type mockIndexer struct {
	indexed []ace.ProductSyncRequest
	deleted []string
	err     error
}

func (m *mockIndexer) IndexProduct(_ context.Context, _, _ string, p ace.ProductSyncRequest) error {
	if m.err != nil {
		return m.err
	}
	m.indexed = append(m.indexed, p)
	return nil
}

func (m *mockIndexer) DeleteProduct(_ context.Context, _, productID string) error {
	if m.err != nil {
		return m.err
	}
	m.deleted = append(m.deleted, productID)
	return nil
}

type mockResolver struct {
	storeID   string
	storeName string
	ok        bool
}

func (m *mockResolver) ResolveToken(_ string) (string, string, bool) {
	return m.storeID, m.storeName, m.ok
}

// --- Tests ---

func TestSyncProducts_Unauthorized(t *testing.T) {
	h := NewSyncHandler(&mockIndexer{}, &mockResolver{ok: false})

	body := `{"product_id":"p1","name":"Widget"}`
	req := httptest.NewRequest(http.MethodPost, "/registry/v1/products/sync", bytes.NewBufferString(body))
	// No Authorization header.
	w := httptest.NewRecorder()

	h.SyncProducts(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	var errResp ace.ErrorResponse
	json.NewDecoder(w.Body).Decode(&errResp)
	if errResp.Code != "unauthorized" {
		t.Fatalf("expected code 'unauthorized', got %q", errResp.Code)
	}
}

func TestSyncProducts_SingleProduct(t *testing.T) {
	idx := &mockIndexer{}
	resolver := &mockResolver{storeID: "str_abc", storeName: "Test Store", ok: true}
	h := NewSyncHandler(idx, resolver)

	product := ace.ProductSyncRequest{
		ProductID:   "p1",
		Name:        "Widget",
		Description: "A fine widget",
		Category:    "tools",
		InStock:     true,
		PriceRange:  ace.PriceRange{Min: 100, Max: 200, Currency: "USD"},
		Rating:      ace.Rating{Average: 4.5, Count: 10},
		Location:    ace.Location{Country: "US", Region: "CA"},
	}

	body, _ := json.Marshal(product)
	req := httptest.NewRequest(http.MethodPost, "/registry/v1/products/sync", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token-123")
	w := httptest.NewRecorder()

	h.SyncProducts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ace.SyncResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Indexed != 1 {
		t.Fatalf("expected 1 indexed, got %d", resp.Indexed)
	}
	if len(resp.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(resp.Errors))
	}
	if len(idx.indexed) != 1 {
		t.Fatalf("expected indexer to receive 1 product, got %d", len(idx.indexed))
	}
	if idx.indexed[0].ProductID != "p1" {
		t.Fatalf("expected product_id 'p1', got %q", idx.indexed[0].ProductID)
	}
}

func TestSyncProducts_BatchProducts(t *testing.T) {
	idx := &mockIndexer{}
	resolver := &mockResolver{storeID: "str_abc", storeName: "Test Store", ok: true}
	h := NewSyncHandler(idx, resolver)

	batch := ace.ProductBatchSyncRequest{
		Products: []ace.ProductSyncRequest{
			{ProductID: "p1", Name: "Widget 1", Category: "tools", InStock: true,
				PriceRange: ace.PriceRange{Min: 100, Max: 200, Currency: "USD"},
				Rating:     ace.Rating{Average: 4.0, Count: 5},
				Location:   ace.Location{Country: "US", Region: "NY"}},
			{ProductID: "p2", Name: "Widget 2", Category: "tools", InStock: false,
				PriceRange: ace.PriceRange{Min: 300, Max: 400, Currency: "USD"},
				Rating:     ace.Rating{Average: 3.5, Count: 8},
				Location:   ace.Location{Country: "US", Region: "CA"}},
		},
	}

	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/registry/v1/products/sync", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token-123")
	w := httptest.NewRecorder()

	h.SyncProducts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ace.SyncResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Indexed != 2 {
		t.Fatalf("expected 2 indexed, got %d", resp.Indexed)
	}
	if len(idx.indexed) != 2 {
		t.Fatalf("expected indexer to receive 2 products, got %d", len(idx.indexed))
	}
}

func TestDeleteSyncedProduct_Unauthorized(t *testing.T) {
	h := NewSyncHandler(&mockIndexer{}, &mockResolver{ok: false})

	req := httptest.NewRequest(http.MethodDelete, "/registry/v1/products/sync/p1", nil)
	req.SetPathValue("product_id", "p1")
	w := httptest.NewRecorder()

	h.DeleteSyncedProduct(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDeleteSyncedProduct_Success(t *testing.T) {
	idx := &mockIndexer{}
	resolver := &mockResolver{storeID: "str_abc", storeName: "Test Store", ok: true}
	h := NewSyncHandler(idx, resolver)

	req := httptest.NewRequest(http.MethodDelete, "/registry/v1/products/sync/p1", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")
	req.SetPathValue("product_id", "p1")
	w := httptest.NewRecorder()

	h.DeleteSyncedProduct(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if len(idx.deleted) != 1 || idx.deleted[0] != "p1" {
		t.Fatalf("expected delete of 'p1', got %v", idx.deleted)
	}
}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"valid bearer", "Bearer abc123", "abc123"},
		{"no header", "", ""},
		{"wrong scheme", "Basic abc123", ""},
		{"bearer only", "Bearer ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			got := extractToken(req)
			if got != tt.want {
				t.Errorf("extractToken() = %q, want %q", got, tt.want)
			}
		})
	}
}
