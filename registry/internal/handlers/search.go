package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/nicroldan/ans/registry/internal/search"
	"github.com/nicroldan/ans/shared/ace"
)

// Searcher abstracts the product search engine.
type Searcher interface {
	Search(ctx context.Context, params search.SearchParams) ([]ace.ProductSearchResult, int, error)
}

// SearchHandler handles product search HTTP endpoints.
type SearchHandler struct {
	searcher Searcher
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(s Searcher) *SearchHandler {
	return &SearchHandler{searcher: s}
}

// RegisterRoutes registers search routes on the given mux.
func (h *SearchHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /registry/v1/search", h.Search)
}

// Search handles GET /registry/v1/search.
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	priceMin, _ := strconv.ParseInt(q.Get("price_min"), 10, 64)
	priceMax, _ := strconv.ParseInt(q.Get("price_max"), 10, 64)

	// Default in_stock to true unless explicitly set to "false".
	var inStock *bool
	inStockParam := q.Get("in_stock")
	switch inStockParam {
	case "false":
		v := false
		inStock = &v
	case "":
		// Default: filter to in-stock products only.
		v := true
		inStock = &v
	default:
		v := true
		inStock = &v
	}

	params := search.SearchParams{
		Query:    q.Get("q"),
		Category: q.Get("category"),
		Country:  q.Get("country"),
		Currency: q.Get("currency"),
		PriceMin: priceMin,
		PriceMax: priceMax,
		InStock:  inStock,
		Sort:     q.Get("sort"),
		Offset:   offset,
		Limit:    limit,
	}

	results, total, err := h.searcher.Search(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search_error", "Search failed")
		return
	}

	writeJSON(w, http.StatusOK, ace.PaginatedResponse[ace.ProductSearchResult]{
		Data:   results,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	})
}
