# Scalable Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add centralized product search to the ACE Protocol registry using Elasticsearch, with a push-based sync mechanism for stores.

**Architecture:** Stores push lightweight product metadata to the registry via `POST /registry/v1/products/sync` using a `registry_token`. The registry indexes products in Elasticsearch. Buyers search via `GET /registry/v1/search` with full-text + filters. The registry falls back to in-memory storage when Elasticsearch is unavailable.

**Tech Stack:** Go 1.25, Elasticsearch 8.x (via `olivere/elastic/v8` or official `elastic/go-elasticsearch`), Docker Compose, stdlib `net/http`.

**Spec:** `docs/superpowers/specs/2026-03-21-scalable-search-design.md`

---

## File Structure

```
CLI Marketplace/
├── docker-compose.yml                          # NEW — ES + registry + ace-server
├── shared/ace/
│   └── types.go                                # MODIFY — add search types
├── registry/
│   ├── go.mod                                  # MODIFY — add ES dependency
│   ├── cmd/registry/
│   │   └── main.go                             # MODIFY — init ES, token store, new routes
│   ├── internal/
│   │   ├── store/
│   │   │   └── memory.go                       # MODIFY — add token storage
│   │   ├── search/
│   │   │   ├── elasticsearch.go                # NEW — ES client, index management
│   │   │   └── elasticsearch_test.go           # NEW — integration tests
│   │   ├── auth/
│   │   │   └── token.go                        # NEW — registry_token generation + validation
│   │   │   └── token_test.go                   # NEW — unit tests
│   │   └── handlers/
│   │       ├── stores.go                       # MODIFY — return registry_token on create
│   │       ├── search.go                       # NEW — GET /registry/v1/search
│   │       ├── search_test.go                  # NEW — unit tests
│   │       ├── sync.go                         # NEW — POST/DELETE /registry/v1/products/sync
│   │       └── sync_test.go                    # NEW — unit tests
└── ace-server/
    └── internal/
        └── sync/
            ├── registry_sync.go                # NEW — push products to registry
            └── registry_sync_test.go           # NEW — unit tests
```

---

### Task 1: Shared Types for Search

**Files:**
- Modify: `shared/ace/types.go`

- [ ] **Step 1: Add search-related types to shared/ace/types.go**

Add after the existing Registry types section (after line 180):

```go
// --- Search/Sync types ---

// ProductSyncRequest is the request body for pushing a single product to the registry index.
type ProductSyncRequest struct {
	ProductID       string   `json:"product_id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	Tags            []string `json:"tags,omitempty"`
	PriceRange      PriceRange `json:"price_range"`
	VariantsSummary []string `json:"variants_summary,omitempty"`
	ImageURL        string   `json:"image_url,omitempty"`
	InStock         bool     `json:"in_stock"`
	Rating          Rating   `json:"rating"`
	Location        Location `json:"location"`
}

// ProductBatchSyncRequest wraps multiple products for batch sync.
type ProductBatchSyncRequest struct {
	Products []ProductSyncRequest `json:"products"`
}

// PriceRange represents the min/max price for a product across variants.
type PriceRange struct {
	Min      int64  `json:"min"`
	Max      int64  `json:"max"`
	Currency string `json:"currency"`
}

// Rating represents aggregated product ratings.
type Rating struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// Location represents geographic information for search filtering.
type Location struct {
	Country string `json:"country"`
	Region  string `json:"region"`
}

// ProductSearchResult is a single product returned from the search index.
type ProductSearchResult struct {
	ProductID       string   `json:"product_id"`
	StoreID         string   `json:"store_id"`
	StoreName       string   `json:"store_name"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	Tags            []string `json:"tags,omitempty"`
	PriceRange      PriceRange `json:"price_range"`
	VariantsSummary []string `json:"variants_summary,omitempty"`
	ImageURL        string   `json:"image_url,omitempty"`
	InStock         bool     `json:"in_stock"`
	Rating          Rating   `json:"rating"`
	Location        Location `json:"location"`
}

// SyncResponse is the response for product sync operations.
type SyncResponse struct {
	Indexed int          `json:"indexed"`
	Updated int          `json:"updated"`
	Errors  []SyncError  `json:"errors,omitempty"`
}

// SyncError reports a per-product sync failure.
type SyncError struct {
	ProductID string `json:"product_id"`
	Error     string `json:"error"`
}

// StoreRegistrationResponse wraps StoreEntry with the one-time registry token.
type StoreRegistrationResponse struct {
	StoreEntry
	RegistryToken string `json:"registry_token"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/shared" && go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add shared/ace/types.go
git commit -m "feat: add search and sync types to shared ACE types"
```

---

### Task 2: Registry Token Auth

**Files:**
- Create: `registry/internal/auth/token.go`
- Create: `registry/internal/auth/token_test.go`

- [ ] **Step 1: Write the failing test**

Create `registry/internal/auth/token_test.go`:

```go
package auth

import "testing"

func TestGenerateToken(t *testing.T) {
	token := GenerateToken()
	if len(token) < 10 {
		t.Fatal("token too short")
	}
	if token[:4] != "rgt_" {
		t.Fatalf("expected rgt_ prefix, got %s", token[:4])
	}
}

func TestHashAndValidate(t *testing.T) {
	token := GenerateToken()
	hash := HashToken(token)

	if !ValidateToken(token, hash) {
		t.Fatal("expected valid token")
	}
	if ValidateToken("wrong_token", hash) {
		t.Fatal("expected invalid token")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go test ./internal/auth/ -v`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Write minimal implementation**

Create `registry/internal/auth/token.go`:

```go
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateToken creates a new registry token with "rgt_" prefix.
func GenerateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "rgt_" + hex.EncodeToString(b)
}

// HashToken returns a SHA-256 hash of the token for storage.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// ValidateToken checks a raw token against a stored hash.
func ValidateToken(token, storedHash string) bool {
	return HashToken(token) == storedHash
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go test ./internal/auth/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add registry/internal/auth/
git commit -m "feat: add registry token generation and validation"
```

---

### Task 3: Token Storage in Registry MemoryStore

**Files:**
- Modify: `registry/internal/store/memory.go`

- [ ] **Step 1: Add token storage to MemoryStore**

In `registry/internal/store/memory.go`, modify the `MemoryStore` struct and methods:

Add a `tokenHashes` map to the struct (mapping `store_id` → `token_hash`):

```go
type MemoryStore struct {
	mu          sync.RWMutex
	entries     map[string]ace.StoreEntry
	tokenHashes map[string]string // store_id → token_hash
}
```

Update `New()`:

```go
func New() *MemoryStore {
	return &MemoryStore{
		entries:     make(map[string]ace.StoreEntry),
		tokenHashes: make(map[string]string),
	}
}
```

Add three new methods:

```go
// StoreTokenHash saves the token hash for a store.
func (m *MemoryStore) StoreTokenHash(storeID, hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokenHashes[storeID] = hash
}

// GetStoreIDByTokenHash returns the store ID for a given token hash.
func (m *MemoryStore) GetStoreIDByTokenHash(hash string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for id, h := range m.tokenHashes {
		if h == hash {
			return id, true
		}
	}
	return "", false
}

// DeleteStore removes a store and its token hash.
func (m *MemoryStore) DeleteStore(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.entries[id]; !ok {
		return false
	}
	delete(m.entries, id)
	delete(m.tokenHashes, id)
	return true
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add registry/internal/store/memory.go
git commit -m "feat: add token hash storage to registry MemoryStore"
```

---

### Task 4: Elasticsearch Client

**Files:**
- Modify: `registry/go.mod`
- Create: `registry/internal/search/elasticsearch.go`
- Create: `registry/internal/search/elasticsearch_test.go`

- [ ] **Step 1: Add Elasticsearch dependency**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go get github.com/elastic/go-elasticsearch/v8`

- [ ] **Step 2: Create the Elasticsearch client wrapper**

Create `registry/internal/search/elasticsearch.go`:

```go
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
```

- [ ] **Step 3: Verify it compiles**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add registry/go.mod registry/go.sum registry/internal/search/
git commit -m "feat: add Elasticsearch search engine for product index"
```

---

### Task 5: Sync Handler (POST/DELETE /registry/v1/products/sync)

**Files:**
- Create: `registry/internal/handlers/sync.go`
- Create: `registry/internal/handlers/sync_test.go`

- [ ] **Step 1: Write the failing test**

Create `registry/internal/handlers/sync_test.go`:

```go
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicroldan/ans/shared/ace"
)

type mockSearchEngine struct {
	indexed []ace.ProductSyncRequest
	deleted []string
}

func (m *mockSearchEngine) IndexProduct(_ context.Context, storeID, storeName string, p ace.ProductSyncRequest) error {
	m.indexed = append(m.indexed, p)
	return nil
}

func (m *mockSearchEngine) DeleteProduct(_ context.Context, storeID, productID string) error {
	m.deleted = append(m.deleted, productID)
	return nil
}

type mockTokenResolver struct {
	storeID   string
	storeName string
}

func (m *mockTokenResolver) ResolveToken(token string) (storeID, storeName string, ok bool) {
	if token == "valid_token" {
		return m.storeID, m.storeName, true
	}
	return "", "", false
}

func TestSyncProduct_Unauthorized(t *testing.T) {
	h := NewSyncHandler(nil, nil)
	req := httptest.NewRequest("POST", "/registry/v1/products/sync", nil)
	w := httptest.NewRecorder()
	h.SyncProducts(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSyncProduct_SingleProduct(t *testing.T) {
	engine := &mockSearchEngine{}
	resolver := &mockTokenResolver{storeID: "str_1", storeName: "Test Store"}
	h := NewSyncHandler(engine, resolver)

	product := ace.ProductSyncRequest{
		ProductID: "prod_1",
		Name:      "Test Product",
		Category:  "electronics",
		PriceRange: ace.PriceRange{Min: 1000, Max: 2000, Currency: "USD"},
		InStock:   true,
		Rating:    ace.Rating{Average: 4.5, Count: 10},
		Location:  ace.Location{Country: "US", Region: "NA"},
	}
	body, _ := json.Marshal(product)
	req := httptest.NewRequest("POST", "/registry/v1/products/sync", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid_token")
	w := httptest.NewRecorder()

	h.SyncProducts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(engine.indexed) != 1 {
		t.Fatalf("expected 1 indexed product, got %d", len(engine.indexed))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go test ./internal/handlers/ -v -run TestSync`
Expected: FAIL — `NewSyncHandler` not found

- [ ] **Step 3: Write the SyncHandler implementation**

Create `registry/internal/handlers/sync.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nicroldan/ans/shared/ace"
)

// ProductIndexer abstracts the search engine for indexing operations.
type ProductIndexer interface {
	IndexProduct(ctx context.Context, storeID, storeName string, p ace.ProductSyncRequest) error
	DeleteProduct(ctx context.Context, storeID, productID string) error
}

// TokenResolver resolves a registry token to a store identity.
type TokenResolver interface {
	ResolveToken(token string) (storeID, storeName string, ok bool)
}

// SyncHandler handles product sync endpoints.
type SyncHandler struct {
	indexer  ProductIndexer
	resolver TokenResolver
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(indexer ProductIndexer, resolver TokenResolver) *SyncHandler {
	return &SyncHandler{indexer: indexer, resolver: resolver}
}

// RegisterRoutes registers sync routes on the given mux.
func (h *SyncHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /registry/v1/products/sync", h.SyncProducts)
	mux.HandleFunc("DELETE /registry/v1/products/sync/{product_id}", h.DeleteSyncedProduct)
}

// extractToken extracts the Bearer token from the Authorization header.
func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// SyncProducts handles POST /registry/v1/products/sync.
func (h *SyncHandler) SyncProducts(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid Authorization header")
		return
	}

	storeID, storeName, ok := h.resolver.ResolveToken(token)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid registry token")
		return
	}

	// Try to decode as batch first, then single product
	var batch ace.ProductBatchSyncRequest
	body := r.Body
	defer body.Close()

	var rawBody json.RawMessage
	if err := json.NewDecoder(body).Decode(&rawBody); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body")
		return
	}

	var products []ace.ProductSyncRequest

	if err := json.Unmarshal(rawBody, &batch); err == nil && len(batch.Products) > 0 {
		products = batch.Products
	} else {
		var single ace.ProductSyncRequest
		if err := json.Unmarshal(rawBody, &single); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "Invalid product data")
			return
		}
		products = []ace.ProductSyncRequest{single}
	}

	resp := ace.SyncResponse{}
	for _, p := range products {
		if p.ProductID == "" || p.Name == "" {
			resp.Errors = append(resp.Errors, ace.SyncError{
				ProductID: p.ProductID,
				Error:     "missing required field: product_id and name are required",
			})
			continue
		}

		err := h.indexer.IndexProduct(r.Context(), storeID, storeName, p)
		if err != nil {
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
	token := extractToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing Authorization header")
		return
	}

	storeID, _, ok := h.resolver.ResolveToken(token)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_token", "Invalid registry token")
		return
	}

	productID := r.PathValue("product_id")
	if err := h.indexer.DeleteProduct(r.Context(), storeID, productID); err != nil {
		writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Fix test imports and run**

Add `"context"` import to sync_test.go, then run:

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go test ./internal/handlers/ -v -run TestSync`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add registry/internal/handlers/sync.go registry/internal/handlers/sync_test.go
git commit -m "feat: add product sync handler with token auth"
```

---

### Task 6: Search Handler (GET /registry/v1/search)

**Files:**
- Create: `registry/internal/handlers/search.go`
- Create: `registry/internal/handlers/search_test.go`

- [ ] **Step 1: Write the failing test**

Create `registry/internal/handlers/search_test.go`:

```go
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicroldan/ans/shared/ace"
	"github.com/nicroldan/ans/registry/internal/search"
)

type mockSearcher struct{}

func (m *mockSearcher) Search(_ context.Context, params search.SearchParams) ([]ace.ProductSearchResult, int, error) {
	return []ace.ProductSearchResult{
		{
			ProductID: "prod_1",
			StoreID:   "str_1",
			StoreName: "Test Store",
			Name:      "Test Product",
			Category:  "electronics",
			PriceRange: ace.PriceRange{Min: 1000, Max: 2000, Currency: "USD"},
			InStock:    true,
			Rating:     ace.Rating{Average: 4.5, Count: 10},
			Location:   ace.Location{Country: "US", Region: "NA"},
		},
	}, 1, nil
}

func TestSearch_Basic(t *testing.T) {
	h := NewSearchHandler(&mockSearcher{})
	req := httptest.NewRequest("GET", "/registry/v1/search?q=test", nil)
	w := httptest.NewRecorder()

	h.Search(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestSearch_DefaultInStock(t *testing.T) {
	h := NewSearchHandler(&mockSearcher{})
	req := httptest.NewRequest("GET", "/registry/v1/search?q=test", nil)
	w := httptest.NewRecorder()

	h.Search(w, req)

	// Default in_stock=true is applied by the handler
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go test ./internal/handlers/ -v -run TestSearch`
Expected: FAIL — `NewSearchHandler` not found

- [ ] **Step 3: Write the SearchHandler implementation**

Create `registry/internal/handlers/search.go`:

```go
package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/nicroldan/ans/shared/ace"
	"github.com/nicroldan/ans/registry/internal/search"
)

// Searcher abstracts the search engine for query operations.
type Searcher interface {
	Search(ctx context.Context, params search.SearchParams) ([]ace.ProductSearchResult, int, error)
}

// SearchHandler handles the search endpoint.
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
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	priceMin, _ := strconv.ParseInt(q.Get("price_min"), 10, 64)
	priceMax, _ := strconv.ParseInt(q.Get("price_max"), 10, 64)

	// Default: in_stock = true
	inStock := true
	if q.Get("in_stock") == "false" {
		inStock = false
	}

	params := search.SearchParams{
		Query:    q.Get("q"),
		Category: q.Get("category"),
		Country:  q.Get("country"),
		Currency: q.Get("currency"),
		PriceMin: priceMin,
		PriceMax: priceMax,
		InStock:  &inStock,
		Sort:     q.Get("sort"),
		Offset:   offset,
		Limit:    limit,
	}

	results, total, err := h.searcher.Search(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ace.PaginatedResponse[ace.ProductSearchResult]{
		Data:   results,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	})
}
```

- [ ] **Step 4: Run tests**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go test ./internal/handlers/ -v -run TestSearch`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add registry/internal/handlers/search.go registry/internal/handlers/search_test.go
git commit -m "feat: add search handler with full-text and filter support"
```

---

### Task 7: Modify Store Registration to Return Token

**Files:**
- Modify: `registry/internal/handlers/stores.go`

- [ ] **Step 1: Modify CreateStore to generate and return token**

In `registry/internal/handlers/stores.go`, update the `StoreHandler` struct to accept a token store, and modify `CreateStore`:

Update the struct and constructor:

```go
type StoreHandler struct {
	store *store.MemoryStore
}
```

becomes:

```go
import (
	"github.com/nicroldan/ans/registry/internal/auth"
)

type StoreHandler struct {
	store *store.MemoryStore
}
```

Modify `CreateStore` to generate a token, store its hash, and return `StoreRegistrationResponse`:

Replace the last two lines of `CreateStore` (lines 65-66: `created := h.store.Create(entry)` and `writeJSON(w, http.StatusCreated, created)`) with:

```go
	created := h.store.Create(entry)

	// Generate registry token for this store.
	token := auth.GenerateToken()
	hash := auth.HashToken(token)
	h.store.StoreTokenHash(created.ID, hash)

	writeJSON(w, http.StatusCreated, ace.StoreRegistrationResponse{
		StoreEntry:    created,
		RegistryToken: token,
	})
```

- [ ] **Step 2: Verify it compiles**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add registry/internal/handlers/stores.go
git commit -m "feat: return registry_token on store registration"
```

---

### Task 8: Wire Everything in Registry main.go

**Files:**
- Modify: `registry/cmd/registry/main.go`

- [ ] **Step 1: Add a TokenResolver implementation that bridges MemoryStore to SyncHandler**

Add to `registry/internal/store/memory.go`:

```go
// ResolveToken finds the store ID and name for a given raw token.
func (m *MemoryStore) ResolveToken(rawToken string) (storeID, storeName string, ok bool) {
	hash := auth.HashToken(rawToken) // need to import auth package
	id, found := m.GetStoreIDByTokenHash(hash)
	if !found {
		return "", "", false
	}
	entry, exists := m.GetByID(id)
	if !exists {
		return "", "", false
	}
	return id, entry.Name, true
}
```

Note: To avoid circular imports, use the hash function inline instead:

```go
import (
	"crypto/sha256"
	"encoding/hex"
)

// ResolveToken finds the store ID and name for a given raw token.
func (m *MemoryStore) ResolveToken(rawToken string) (storeID, storeName string, ok bool) {
	h := sha256.Sum256([]byte(rawToken))
	hash := hex.EncodeToString(h[:])
	id, found := m.GetStoreIDByTokenHash(hash)
	if !found {
		return "", "", false
	}
	m.mu.RLock()
	entry, exists := m.entries[id]
	m.mu.RUnlock()
	if !exists {
		return "", "", false
	}
	return id, entry.Name, true
}
```

- [ ] **Step 2: Update registry main.go to initialize ES and register new routes**

Replace the content of `registry/cmd/registry/main.go`:

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nicroldan/ans/registry/internal/handlers"
	"github.com/nicroldan/ans/registry/internal/search"
	"github.com/nicroldan/ans/registry/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		esURL = "http://localhost:9200"
	}

	memStore := store.New()
	mux := http.NewServeMux()

	// Store handler (existing).
	storeHandler := handlers.NewStoreHandler(memStore)
	storeHandler.RegisterRoutes(mux)

	// Elasticsearch search engine.
	engine, err := search.NewEngine([]string{esURL})
	if err != nil {
		log.Printf("WARNING: Elasticsearch unavailable (%v) — search disabled", err)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := engine.Ping(ctx); err != nil {
			log.Printf("WARNING: Elasticsearch not reachable (%v) — search disabled", err)
			engine = nil
		} else {
			if err := engine.EnsureIndex(ctx); err != nil {
				log.Fatalf("Failed to create ES index: %v", err)
			}
			log.Printf("Elasticsearch connected at %s", esURL)
		}
	}

	if engine != nil {
		// Sync handler.
		syncHandler := handlers.NewSyncHandler(engine, memStore)
		syncHandler.RegisterRoutes(mux)

		// Search handler.
		searchHandler := handlers.NewSearchHandler(engine)
		searchHandler.RegisterRoutes(mux)
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Registry server listening on :%s", port)
		if engine != nil {
			log.Printf("Search enabled (Elasticsearch)")
		} else {
			log.Printf("Search disabled (no Elasticsearch)")
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
```

- [ ] **Step 3: Fix interface compliance**

The `search.Engine` needs to satisfy both `ProductIndexer` and `Searcher` interfaces. Verify that the method signatures match. Since `NewSyncHandler` takes `ProductIndexer` and `NewSearchHandler` takes `Searcher`, and `*search.Engine` implements both, this should compile.

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add registry/
git commit -m "feat: wire Elasticsearch, sync, and search into registry"
```

---

### Task 9: Docker Compose

**Files:**
- Create: `docker-compose.yml`

- [ ] **Step 1: Create docker-compose.yml**

Create `docker-compose.yml` at the project root:

```yaml
version: "3.8"

services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.17.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    ports:
      - "9200:9200"
    volumes:
      - es_data:/usr/share/elasticsearch/data
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:9200/_cluster/health || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  registry:
    build:
      context: .
      dockerfile: registry/Dockerfile
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - ELASTICSEARCH_URL=http://elasticsearch:9200
    depends_on:
      elasticsearch:
        condition: service_healthy

  ace-server:
    build:
      context: .
      dockerfile: ace-server/Dockerfile
    ports:
      - "8081:8081"
    environment:
      - PORT=8081
      - STORE_NAME=ACE Demo Store
      - BASE_URL=http://localhost:8081

volumes:
  es_data:
```

- [ ] **Step 2: Create Dockerfiles**

Create `registry/Dockerfile`:

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work
COPY shared/ shared/
COPY registry/ registry/
RUN cd registry && go build -o /registry ./cmd/registry

FROM alpine:3.21
COPY --from=builder /registry /registry
CMD ["/registry"]
```

Create `ace-server/Dockerfile`:

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work
COPY shared/ shared/
COPY ace-server/ ace-server/
RUN cd ace-server && go build -o /ace-server ./cmd/ace-server

FROM alpine:3.21
COPY --from=builder /ace-server /ace-server
CMD ["/ace-server"]
```

- [ ] **Step 3: Test that Elasticsearch starts**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace" && docker compose up elasticsearch -d`
Wait for healthy, then: `curl -s http://localhost:9200/_cluster/health | head -1`
Expected: JSON with `"status":"green"` or `"status":"yellow"`

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml registry/Dockerfile ace-server/Dockerfile
git commit -m "feat: add Docker Compose with Elasticsearch for local dev"
```

---

### Task 10: ACE Server Sync Client (Push Products to Registry)

**Files:**
- Create: `ace-server/internal/sync/registry_sync.go`
- Create: `ace-server/internal/sync/registry_sync_test.go`

- [ ] **Step 1: Write the failing test**

Create `ace-server/internal/sync/registry_sync_test.go`:

```go
package sync

import (
	"testing"

	"github.com/nicroldan/ans/shared/ace"
)

func TestBuildSyncRequest_NoVariants(t *testing.T) {
	p := &ace.Product{
		ID:          "prod_1",
		Name:        "Test",
		Description: "Desc",
		Price:       ace.Money{Amount: 1000, Currency: "USD"},
	}
	req := BuildSyncRequest(p, "electronics", []string{"test"}, ace.Location{Country: "US", Region: "NA"})
	if req.PriceRange.Min != 1000 || req.PriceRange.Max != 1000 {
		t.Fatalf("expected min=max=1000, got min=%d max=%d", req.PriceRange.Min, req.PriceRange.Max)
	}
	if !req.InStock {
		t.Fatal("product with no variants should be in_stock=true")
	}
}

func TestBuildSyncRequest_WithVariants(t *testing.T) {
	p := &ace.Product{
		ID:          "prod_1",
		Name:        "Test",
		Description: "Desc",
		Price:       ace.Money{Amount: 1000, Currency: "USD"},
		Variants: []ace.Variant{
			{Name: "Small", Price: ace.Money{Amount: 800, Currency: "USD"}, Inventory: 0},
			{Name: "Large", Price: ace.Money{Amount: 1200, Currency: "USD"}, Inventory: 5},
		},
	}
	req := BuildSyncRequest(p, "electronics", nil, ace.Location{Country: "US", Region: "NA"})
	if req.PriceRange.Min != 800 || req.PriceRange.Max != 1200 {
		t.Fatalf("expected min=800 max=1200, got min=%d max=%d", req.PriceRange.Min, req.PriceRange.Max)
	}
	if !req.InStock {
		t.Fatal("product with at least one variant in stock should be in_stock=true")
	}
	if len(req.VariantsSummary) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(req.VariantsSummary))
	}
}

func TestBuildSyncRequest_AllOutOfStock(t *testing.T) {
	p := &ace.Product{
		ID:   "prod_1",
		Name: "Test",
		Price: ace.Money{Amount: 1000, Currency: "USD"},
		Variants: []ace.Variant{
			{Name: "Small", Price: ace.Money{Amount: 800, Currency: "USD"}, Inventory: 0},
			{Name: "Large", Price: ace.Money{Amount: 1200, Currency: "USD"}, Inventory: 0},
		},
	}
	req := BuildSyncRequest(p, "electronics", nil, ace.Location{Country: "US", Region: "NA"})
	if req.InStock {
		t.Fatal("product with all variants out of stock should be in_stock=false")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-server" && go test ./internal/sync/ -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Write the implementation**

Create `ace-server/internal/sync/registry_sync.go`:

```go
package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nicroldan/ans/shared/ace"
)

// BuildSyncRequest converts an ace.Product into a ProductSyncRequest for the registry.
func BuildSyncRequest(p *ace.Product, category string, tags []string, location ace.Location) ace.ProductSyncRequest {
	priceRange := ace.PriceRange{
		Min:      p.Price.Amount,
		Max:      p.Price.Amount,
		Currency: p.Price.Currency,
	}

	var variantNames []string
	inStock := true // no variants = always in stock

	if len(p.Variants) > 0 {
		inStock = false
		priceRange.Min = p.Variants[0].Price.Amount
		priceRange.Max = p.Variants[0].Price.Amount

		for _, v := range p.Variants {
			if v.Price.Amount < priceRange.Min {
				priceRange.Min = v.Price.Amount
			}
			if v.Price.Amount > priceRange.Max {
				priceRange.Max = v.Price.Amount
			}
			if v.Inventory > 0 {
				inStock = true
			}
			variantNames = append(variantNames, v.Name)
		}
	}

	return ace.ProductSyncRequest{
		ProductID:       p.ID,
		Name:            p.Name,
		Description:     p.Description,
		Category:        category,
		Tags:            tags,
		PriceRange:      priceRange,
		VariantsSummary: variantNames,
		InStock:         inStock,
		Location:        location,
	}
}

// Client pushes product data to the registry's sync endpoint.
type Client struct {
	registryURL   string
	registryToken string
	httpClient    *http.Client
}

// NewClient creates a new sync client.
func NewClient(registryURL, registryToken string) *Client {
	return &Client{
		registryURL:   registryURL,
		registryToken: registryToken,
		httpClient:    &http.Client{},
	}
}

// PushProduct syncs a single product to the registry.
func (c *Client) PushProduct(req ace.ProductSyncRequest) (*ace.SyncResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.registryURL+"/registry/v1/products/sync", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.registryToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sync failed with status %d", resp.StatusCode)
	}

	var syncResp ace.SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return nil, err
	}
	return &syncResp, nil
}

// DeleteProduct removes a product from the registry index.
func (c *Client) DeleteProduct(productID string) error {
	httpReq, err := http.NewRequest("DELETE", c.registryURL+"/registry/v1/products/sync/"+productID, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.registryToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("delete sync request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("delete sync failed with status %d", resp.StatusCode)
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-server" && go test ./internal/sync/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add ace-server/internal/sync/
git commit -m "feat: add registry sync client for ace-server product push"
```

---

### Task 11: Update Agent Buyer to Use Search

**Files:**
- Modify: `agent-buyer/internal/client/registry.go`
- Modify: `agent-buyer/cmd/buyer/main.go`

- [ ] **Step 1: Add search method to registry client**

Read `agent-buyer/internal/client/registry.go` and add a `SearchProducts` method:

```go
// SearchProducts searches for products across all stores.
func (c *RegistryClient) SearchProducts(query, country, category string) ([]ace.ProductSearchResult, error) {
	u := fmt.Sprintf("%s/registry/v1/search?q=%s", c.baseURL, url.QueryEscape(query))
	if country != "" {
		u += "&country=" + url.QueryEscape(country)
	}
	if category != "" {
		u += "&category=" + url.QueryEscape(category)
	}

	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ace.PaginatedResponse[ace.ProductSearchResult]
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
```

- [ ] **Step 2: Update buyer main.go to include a search step**

Add an optional search step at the start of the buyer flow (after store discovery) that searches for products via the registry before connecting to a specific store.

- [ ] **Step 3: Verify it compiles**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/agent-buyer" && go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add agent-buyer/
git commit -m "feat: add product search to buyer agent via registry"
```

---

### Task 12: End-to-End Manual Test

- [ ] **Step 1: Start Elasticsearch**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace" && docker compose up elasticsearch -d`
Wait until healthy.

- [ ] **Step 2: Start registry**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/registry" && ELASTICSEARCH_URL=http://localhost:9200 go run ./cmd/registry`

- [ ] **Step 3: Register a store and capture token**

```bash
curl -s -X POST http://localhost:8080/registry/v1/stores \
  -H "Content-Type: application/json" \
  -d '{"well_known_url":"http://localhost:8081/.well-known/agent-commerce","categories":["electronics"],"country":"US"}'
```

Save the `registry_token` from the response.

- [ ] **Step 4: Sync a product**

```bash
curl -s -X POST http://localhost:8080/registry/v1/products/sync \
  -H "Authorization: Bearer <REGISTRY_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "product_id":"prod_1",
    "name":"Mechanical Keyboard",
    "description":"Cherry MX switches",
    "category":"electronics",
    "tags":["keyboard","mechanical"],
    "price_range":{"min":6999,"max":8999,"currency":"USD"},
    "variants_summary":["Red","Blue"],
    "in_stock":true,
    "rating":{"average":4.5,"count":100},
    "location":{"country":"US","region":"NA"}
  }'
```

Expected: `{"indexed":1,"updated":0}`

- [ ] **Step 5: Search for the product**

```bash
curl -s "http://localhost:8080/registry/v1/search?q=keyboard&country=US"
```

Expected: paginated response with the keyboard product.

- [ ] **Step 6: Commit any final fixes**

```bash
git add -A
git commit -m "chore: end-to-end verification complete"
```
