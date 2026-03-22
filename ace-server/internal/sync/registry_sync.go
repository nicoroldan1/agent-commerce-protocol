package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/nicroldan/ans/shared/ace"
)

// BuildSyncRequest converts an ace.Product into a ProductSyncRequest suitable
// for pushing to the registry search index. category, tags, and location are
// provided by the caller because they are store-level metadata not present on
// the Product itself.
func BuildSyncRequest(p ace.Product, category string, tags []string, location ace.Location) ace.ProductSyncRequest {
	req := ace.ProductSyncRequest{
		ProductID:   p.ID,
		Name:        p.Name,
		Description: p.Description,
		Category:    category,
		Tags:        tags,
		Location:    location,
	}

	if len(p.Variants) == 0 {
		// No variants: price range is just the product price, always in stock.
		req.PriceRange = ace.PriceRange{
			Min:      p.Price.Amount,
			Max:      p.Price.Amount,
			Currency: p.Price.Currency,
		}
		req.InStock = true
		return req
	}

	// With variants: compute min/max price and stock availability.
	var minPrice int64 = math.MaxInt64
	var maxPrice int64 = math.MinInt64
	var inStock bool
	variantNames := make([]string, 0, len(p.Variants))

	for _, v := range p.Variants {
		if v.Price.Amount < minPrice {
			minPrice = v.Price.Amount
		}
		if v.Price.Amount > maxPrice {
			maxPrice = v.Price.Amount
		}
		if v.Inventory > 0 {
			inStock = true
		}
		variantNames = append(variantNames, v.Name)
	}

	req.PriceRange = ace.PriceRange{
		Min:      minPrice,
		Max:      maxPrice,
		Currency: p.Price.Currency,
	}
	req.InStock = inStock
	req.VariantsSummary = variantNames

	return req
}

// Client is an HTTP client that pushes product data to the registry search index.
type Client struct {
	registryURL   string
	registryToken string
	http          *http.Client
}

// NewClient creates a new registry sync client.
func NewClient(registryURL, registryToken string) *Client {
	return &Client{
		registryURL:   registryURL,
		registryToken: registryToken,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// PushProduct sends a single product to the registry search index.
func (c *Client) PushProduct(req ace.ProductSyncRequest) (*ace.SyncResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal sync request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.registryURL+"/api/v1/sync/products", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.registryToken)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("push product: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registry returned %d: %s", resp.StatusCode, string(respBody))
	}

	var syncResp ace.SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return nil, fmt.Errorf("decode sync response: %w", err)
	}
	return &syncResp, nil
}

// DeleteProduct removes a product from the registry search index.
func (c *Client) DeleteProduct(productID string) error {
	httpReq, err := http.NewRequest(http.MethodDelete, c.registryURL+"/api/v1/sync/products/"+productID, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.registryToken)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
