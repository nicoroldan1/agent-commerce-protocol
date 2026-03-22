package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/nicroldan/ans/shared/ace"
)

const ProductIndex = "ace_products"

// Engine provides search operations on the product index.
type Engine struct {
	client *elasticsearch.Client
}

// NewEngine creates a new search engine connected to Elasticsearch.
func NewEngine(addresses []string) (*Engine, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating ES client: %w", err)
	}
	return &Engine{client: client}, nil
}

// EnsureIndex creates the products index with mappings if it doesn't exist.
func (e *Engine) EnsureIndex(ctx context.Context) error {
	res, err := e.client.Indices.Exists([]string{ProductIndex})
	if err != nil {
		return fmt.Errorf("checking index: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil // already exists
	}

	mapping := `{
		"mappings": {
			"properties": {
				"product_id":       {"type": "keyword"},
				"store_id":         {"type": "keyword"},
				"store_name":       {"type": "text"},
				"name":             {"type": "text", "analyzer": "standard"},
				"description":      {"type": "text", "analyzer": "standard"},
				"category":         {"type": "keyword"},
				"tags":             {"type": "keyword"},
				"price_range": {
					"properties": {
						"min":      {"type": "long"},
						"max":      {"type": "long"},
						"currency": {"type": "keyword"}
					}
				},
				"variants_summary": {"type": "keyword"},
				"image_url":        {"type": "keyword"},
				"in_stock":         {"type": "boolean"},
				"rating": {
					"properties": {
						"average": {"type": "float"},
						"count":   {"type": "integer"}
					}
				},
				"location": {
					"properties": {
						"country": {"type": "keyword"},
						"region":  {"type": "keyword"}
					}
				},
				"updated_at":       {"type": "date"}
			}
		}
	}`

	res, err = e.client.Indices.Create(
		ProductIndex,
		e.client.Indices.Create.WithBody(strings.NewReader(mapping)),
		e.client.Indices.Create.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("creating index: %s", body)
	}
	return nil
}

// docID returns the composite document ID for a product.
func docID(storeID, productID string) string {
	return storeID + "::" + productID
}

// IndexProduct indexes or updates a single product document.
func (e *Engine) IndexProduct(ctx context.Context, storeID, storeName string, p ace.ProductSyncRequest) error {
	doc := map[string]any{
		"product_id":       p.ProductID,
		"store_id":         storeID,
		"store_name":       storeName,
		"name":             p.Name,
		"description":      p.Description,
		"category":         p.Category,
		"tags":             p.Tags,
		"price_range":      p.PriceRange,
		"variants_summary": p.VariantsSummary,
		"image_url":        p.ImageURL,
		"in_stock":         p.InStock,
		"rating":           p.Rating,
		"location":         p.Location,
		"updated_at":       "now",
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	res, err := e.client.Index(
		ProductIndex,
		bytes.NewReader(body),
		e.client.Index.WithDocumentID(docID(storeID, p.ProductID)),
		e.client.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("indexing product: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("indexing product: %s", body)
	}
	return nil
}

// DeleteProduct removes a product from the index.
func (e *Engine) DeleteProduct(ctx context.Context, storeID, productID string) error {
	res, err := e.client.Delete(
		ProductIndex,
		docID(storeID, productID),
		e.client.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("deleting product: %w", err)
	}
	defer res.Body.Close()
	return nil
}

// DeleteByStore removes all products for a store.
func (e *Engine) DeleteByStore(ctx context.Context, storeID string) error {
	query := fmt.Sprintf(`{"query":{"term":{"store_id":"%s"}}}`, storeID)
	res, err := e.client.DeleteByQuery(
		[]string{ProductIndex},
		strings.NewReader(query),
		e.client.DeleteByQuery.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("deleting by store: %w", err)
	}
	defer res.Body.Close()
	return nil
}

// SearchParams defines the search query parameters.
type SearchParams struct {
	Query    string
	Category string
	Country  string
	Currency string
	PriceMin int64
	PriceMax int64
	InStock  *bool // nil means no filter
	Sort     string
	Offset   int
	Limit    int
}

// Search executes a product search and returns results with total count.
func (e *Engine) Search(ctx context.Context, params SearchParams) ([]ace.ProductSearchResult, int, error) {
	must := []map[string]any{}
	filter := []map[string]any{}

	if params.Query != "" {
		must = append(must, map[string]any{
			"multi_match": map[string]any{
				"query":  params.Query,
				"fields": []string{"name^3", "description", "tags^2"},
			},
		})
	}

	if params.Category != "" {
		filter = append(filter, map[string]any{
			"term": map[string]any{"category": params.Category},
		})
	}
	if params.Country != "" {
		filter = append(filter, map[string]any{
			"term": map[string]any{"location.country": params.Country},
		})
	}
	if params.Currency != "" {
		filter = append(filter, map[string]any{
			"term": map[string]any{"price_range.currency": params.Currency},
		})
	}
	if params.InStock != nil {
		filter = append(filter, map[string]any{
			"term": map[string]any{"in_stock": *params.InStock},
		})
	}

	priceRange := map[string]any{}
	if params.PriceMin > 0 {
		priceRange["gte"] = params.PriceMin
	}
	if params.PriceMax > 0 {
		priceRange["lte"] = params.PriceMax
	}
	if len(priceRange) > 0 {
		filter = append(filter, map[string]any{
			"range": map[string]any{"price_range.min": priceRange},
		})
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must":   must,
				"filter": filter,
			},
		},
		"from": params.Offset,
		"size": params.Limit,
	}

	// Sorting
	switch params.Sort {
	case "price_asc":
		query["sort"] = []map[string]any{{"price_range.min": "asc"}}
	case "price_desc":
		query["sort"] = []map[string]any{{"price_range.max": "desc"}}
	case "rating":
		query["sort"] = []map[string]any{{"rating.average": "desc"}}
	default:
		// relevance — no explicit sort needed, ES uses _score
	}

	body, _ := json.Marshal(query)

	res, err := e.client.Search(
		e.client.Search.WithContext(ctx),
		e.client.Search.WithIndex(ProductIndex),
		e.client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		respBody, _ := io.ReadAll(res.Body)
		return nil, 0, fmt.Errorf("search error: %s", respBody)
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source ace.ProductSearchResult `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("decoding search response: %w", err)
	}

	products := make([]ace.ProductSearchResult, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		products = append(products, hit.Source)
	}

	return products, result.Hits.Total.Value, nil
}

// Ping checks if Elasticsearch is reachable.
func (e *Engine) Ping(ctx context.Context) error {
	res, err := e.client.Ping(e.client.Ping.WithContext(ctx))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("ES ping failed: %s", res.Status())
	}
	return nil
}
