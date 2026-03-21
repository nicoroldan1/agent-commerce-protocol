package healthcheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nicroldan/ans/shared/ace"
)

var client = &http.Client{Timeout: 5 * time.Second}

// FetchWellKnown fetches and parses the ACE well-known response from the given URL.
func FetchWellKnown(wellKnownURL string) (ace.WellKnownResponse, error) {
	resp, err := client.Get(wellKnownURL)
	if err != nil {
		return ace.WellKnownResponse{}, fmt.Errorf("failed to reach well-known URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ace.WellKnownResponse{}, fmt.Errorf("well-known URL returned status %d", resp.StatusCode)
	}

	var wk ace.WellKnownResponse
	if err := json.NewDecoder(resp.Body).Decode(&wk); err != nil {
		return ace.WellKnownResponse{}, fmt.Errorf("invalid well-known response: %w", err)
	}

	if wk.Name == "" {
		return ace.WellKnownResponse{}, fmt.Errorf("well-known response missing store name")
	}

	return wk, nil
}
