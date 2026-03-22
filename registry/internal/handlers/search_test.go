package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicroldan/ans/registry/internal/search"
	"github.com/nicroldan/ans/shared/ace"
)

// mockSearcher is a test double that records the params it received and
// returns a fixed result set.
type mockSearcher struct {
	lastParams search.SearchParams
	results    []ace.ProductSearchResult
	total      int
	err        error
}

func (m *mockSearcher) Search(_ context.Context, params search.SearchParams) ([]ace.ProductSearchResult, int, error) {
	m.lastParams = params
	return m.results, m.total, m.err
}

func sampleResults() []ace.ProductSearchResult {
	return []ace.ProductSearchResult{
		{
			ProductID: "prod-1",
			StoreID:   "store-1",
			StoreName: "Test Store",
			Name:      "Widget",
			Category:  "tools",
			InStock:   true,
			PriceRange: ace.PriceRange{
				Min:      1000,
				Max:      2000,
				Currency: "USD",
			},
			Rating:   ace.Rating{Average: 4.5, Count: 10},
			Location: ace.Location{Country: "US", Region: "CA"},
		},
	}
}

func TestSearch_BasicQuery(t *testing.T) {
	mock := &mockSearcher{
		results: sampleResults(),
		total:   1,
	}

	h := NewSearchHandler(mock)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/registry/v1/search?q=widget&category=tools&limit=10", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp ace.PaginatedResponse[ace.ProductSearchResult]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Data))
	}
	if resp.Data[0].ProductID != "prod-1" {
		t.Errorf("expected prod-1, got %s", resp.Data[0].ProductID)
	}
	if resp.Limit != 10 {
		t.Errorf("expected limit 10, got %d", resp.Limit)
	}

	// Verify params passed to searcher.
	if mock.lastParams.Query != "widget" {
		t.Errorf("expected query 'widget', got %q", mock.lastParams.Query)
	}
	if mock.lastParams.Category != "tools" {
		t.Errorf("expected category 'tools', got %q", mock.lastParams.Category)
	}
	if mock.lastParams.Limit != 10 {
		t.Errorf("expected limit 10, got %d", mock.lastParams.Limit)
	}
}

func TestSearch_DefaultInStockTrue(t *testing.T) {
	mock := &mockSearcher{
		results: []ace.ProductSearchResult{},
		total:   0,
	}

	h := NewSearchHandler(mock)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// No in_stock param -- should default to true.
	req := httptest.NewRequest("GET", "/registry/v1/search?q=anything", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if mock.lastParams.InStock == nil {
		t.Fatal("expected InStock to be non-nil (defaulted to true)")
	}
	if *mock.lastParams.InStock != true {
		t.Errorf("expected InStock=true, got %v", *mock.lastParams.InStock)
	}
}

func TestSearch_InStockExplicitFalse(t *testing.T) {
	mock := &mockSearcher{
		results: []ace.ProductSearchResult{},
		total:   0,
	}

	h := NewSearchHandler(mock)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/registry/v1/search?in_stock=false", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if mock.lastParams.InStock == nil {
		t.Fatal("expected InStock to be non-nil")
	}
	if *mock.lastParams.InStock != false {
		t.Errorf("expected InStock=false, got %v", *mock.lastParams.InStock)
	}
}

func TestSearch_DefaultLimit(t *testing.T) {
	mock := &mockSearcher{
		results: []ace.ProductSearchResult{},
		total:   0,
	}

	h := NewSearchHandler(mock)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// No limit param -- should default to 20.
	req := httptest.NewRequest("GET", "/registry/v1/search", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if mock.lastParams.Limit != 20 {
		t.Errorf("expected default limit 20, got %d", mock.lastParams.Limit)
	}
}

func TestSearch_LimitCappedAt100(t *testing.T) {
	mock := &mockSearcher{
		results: []ace.ProductSearchResult{},
		total:   0,
	}

	h := NewSearchHandler(mock)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/registry/v1/search?limit=500", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if mock.lastParams.Limit != 100 {
		t.Errorf("expected limit capped to 100, got %d", mock.lastParams.Limit)
	}
}
