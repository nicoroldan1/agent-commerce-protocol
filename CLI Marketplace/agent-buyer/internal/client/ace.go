package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/nicroldan/ans/shared/ace"
)

// ACEClient communicates with an ACE-compatible store.
type ACEClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewACEClient creates a new client for an ACE store.
func NewACEClient(baseURL, apiKey string) *ACEClient {
	return &ACEClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Discover fetches the well-known agent-commerce descriptor.
func (c *ACEClient) Discover(wellKnownURL string) (*ace.WellKnownResponse, error) {
	var resp ace.WellKnownResponse
	if err := c.doJSON("GET", wellKnownURL, nil, &resp); err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}
	return &resp, nil
}

// ListProducts queries the product catalog.
func (c *ACEClient) ListProducts(query string, offset, limit int) (*ace.PaginatedResponse[ace.Product], error) {
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	params.Set("offset", strconv.Itoa(offset))
	params.Set("limit", strconv.Itoa(limit))

	u := c.baseURL + "/products?" + params.Encode()
	var resp ace.PaginatedResponse[ace.Product]
	if err := c.doJSON("GET", u, nil, &resp); err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	return &resp, nil
}

// GetProduct fetches a single product by ID.
func (c *ACEClient) GetProduct(id string) (*ace.Product, error) {
	u := c.baseURL + "/products/" + url.PathEscape(id)
	var resp ace.Product
	if err := c.doJSON("GET", u, nil, &resp); err != nil {
		return nil, fmt.Errorf("get product %s: %w", id, err)
	}
	return &resp, nil
}

// CreateCart creates a new shopping cart.
func (c *ACEClient) CreateCart() (*ace.Cart, error) {
	u := c.baseURL + "/cart"
	var resp ace.Cart
	if err := c.doJSON("POST", u, struct{}{}, &resp); err != nil {
		return nil, fmt.Errorf("create cart: %w", err)
	}
	return &resp, nil
}

// AddCartItem adds an item to a cart.
func (c *ACEClient) AddCartItem(cartID string, req ace.AddCartItemRequest) (*ace.Cart, error) {
	u := c.baseURL + "/cart/" + url.PathEscape(cartID) + "/items"
	var resp ace.Cart
	if err := c.doJSON("POST", u, req, &resp); err != nil {
		return nil, fmt.Errorf("add cart item: %w", err)
	}
	return &resp, nil
}

// GetCart fetches a cart by ID.
func (c *ACEClient) GetCart(cartID string) (*ace.Cart, error) {
	u := c.baseURL + "/cart/" + url.PathEscape(cartID)
	var resp ace.Cart
	if err := c.doJSON("GET", u, nil, &resp); err != nil {
		return nil, fmt.Errorf("get cart %s: %w", cartID, err)
	}
	return &resp, nil
}

// CreateOrder creates an order from a cart.
func (c *ACEClient) CreateOrder(cartID string) (*ace.Order, error) {
	u := c.baseURL + "/orders"
	req := ace.CreateOrderRequest{CartID: cartID}
	var resp ace.Order
	if err := c.doJSON("POST", u, req, &resp); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	return &resp, nil
}

// GetOrder fetches an order by ID.
func (c *ACEClient) GetOrder(orderID string) (*ace.Order, error) {
	u := c.baseURL + "/orders/" + url.PathEscape(orderID)
	var resp ace.Order
	if err := c.doJSON("GET", u, nil, &resp); err != nil {
		return nil, fmt.Errorf("get order %s: %w", orderID, err)
	}
	return &resp, nil
}

// Pay initiates a payment for an order.
func (c *ACEClient) Pay(orderID, provider string) (*ace.Payment, error) {
	u := c.baseURL + "/orders/" + url.PathEscape(orderID) + "/pay"
	req := ace.InitiatePaymentRequest{Provider: provider}
	var resp ace.Payment
	if err := c.doJSON("POST", u, req, &resp); err != nil {
		return nil, fmt.Errorf("pay order %s: %w", orderID, err)
	}
	return &resp, nil
}

// PaymentStatus checks the payment status for an order.
func (c *ACEClient) PaymentStatus(orderID string) (*ace.Payment, error) {
	u := c.baseURL + "/orders/" + url.PathEscape(orderID) + "/pay/status"
	var resp ace.Payment
	if err := c.doJSON("GET", u, nil, &resp); err != nil {
		return nil, fmt.Errorf("payment status %s: %w", orderID, err)
	}
	return &resp, nil
}

// doJSON performs an HTTP request. For non-nil body it marshals JSON; it unmarshals
// the response into out. Non-2xx responses are parsed as ace.ErrorResponse.
func (c *ACEClient) doJSON(method, rawURL string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("X-ACE-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ace.ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("API error %d: %s (code=%s, details=%s)",
				resp.StatusCode, errResp.Error, errResp.Code, errResp.Details)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}
