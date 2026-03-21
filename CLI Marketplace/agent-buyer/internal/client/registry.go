package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/nicroldan/ans/shared/ace"
)

// RegistryClient communicates with the ANS store registry.
type RegistryClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRegistryClient creates a new registry client.
func NewRegistryClient(baseURL string) *RegistryClient {
	return &RegistryClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SearchStores queries the registry for stores matching the given filters.
func (r *RegistryClient) SearchStores(query, category, country string) (*ace.PaginatedResponse[ace.StoreEntry], error) {
	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if category != "" {
		params.Set("category", category)
	}
	if country != "" {
		params.Set("country", country)
	}

	u := r.baseURL + "/registry/v1/stores"
	if encoded := params.Encode(); encoded != "" {
		u += "?" + encoded
	}

	var resp ace.PaginatedResponse[ace.StoreEntry]
	if err := r.doGet(u, &resp); err != nil {
		return nil, fmt.Errorf("search stores: %w", err)
	}
	return &resp, nil
}

// GetStore fetches a single store entry by ID.
func (r *RegistryClient) GetStore(id string) (*ace.StoreEntry, error) {
	u := r.baseURL + "/registry/v1/stores/" + url.PathEscape(id)
	var resp ace.StoreEntry
	if err := r.doGet(u, &resp); err != nil {
		return nil, fmt.Errorf("get store %s: %w", id, err)
	}
	return &resp, nil
}

func (r *RegistryClient) doGet(rawURL string, out any) error {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ace.ErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("API error %d: %s (code=%s)", resp.StatusCode, errResp.Error, errResp.Code)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}
