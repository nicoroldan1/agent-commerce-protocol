# Headless Merchant Features Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add payment-as-auth (dual mode), per-request pricing, and pricing headers to the ACE Protocol ace-server.

**Architecture:** The auth middleware gains dual mode (API key OR payment token). A PaymentValidator interface with a mock implementation handles token validation. Buyer handlers add pricing headers via a helper. New types in shared/ace support the 402 response and pricing config.

**Tech Stack:** Go 1.25, stdlib `net/http`, existing ace-server patterns.

**Spec:** `docs/superpowers/specs/2026-03-24-headless-merchant-design.md`

---

## File Structure

```
shared/ace/
  types.go                              # MODIFY — add payment/pricing types

ace-server/internal/
  payment/
    validator.go                        # NEW — PaymentValidator interface + MockValidator
    validator_test.go                   # NEW — unit tests
  middleware/
    auth.go                             # MODIFY — dual mode auth (API key OR payment)
    auth_test.go                        # NEW — unit tests for dual mode
  handlers/
    helpers.go                          # MODIFY — add WritePricingHeaders helper
    buyer.go                            # MODIFY — update Discovery, add Pricing endpoint, pricing headers
    buyer_test.go                       # NEW — tests for pricing endpoint

ace-server/cmd/ace-server/
  main.go                              # MODIFY — wire payment validator, env vars, new routes
```

---

### Task 1: Shared Types for Payment and Pricing

**Files:**
- Modify: `shared/ace/types.go`

- [ ] **Step 1: Add payment and pricing types**

Add after the existing search/sync types section:

```go
// --- Payment/Pricing types ---

// PaymentAuthConfig describes payment-as-auth capabilities for a store.
type PaymentAuthConfig struct {
	Enabled         bool     `json:"enabled"`
	Header          string   `json:"header"`
	Providers       []string `json:"providers"`
	DefaultCurrency string   `json:"default_currency"`
}

// PaymentRequiredResponse is returned with HTTP 402 when payment is needed.
type PaymentRequiredResponse struct {
	Error   string      `json:"error"`
	Code    string      `json:"code"`
	Pricing PricingInfo `json:"pricing"`
}

// PricingInfo describes the cost and accepted payment methods for an endpoint.
type PricingInfo struct {
	Price             float64  `json:"price"`
	Currency          string   `json:"currency"`
	AcceptedProviders []string `json:"accepted_providers"`
	DetailsURL        string   `json:"details_url,omitempty"`
}

// PricingSchedule is the response for GET /ace/v1/pricing.
type PricingSchedule struct {
	DefaultCurrency string          `json:"default_currency"`
	Endpoints       []EndpointPrice `json:"endpoints"`
}

// EndpointPrice describes the cost of a single API endpoint.
type EndpointPrice struct {
	Method string  `json:"method"`
	Path   string  `json:"path"`
	Price  float64 `json:"price"`
}
```

- [ ] **Step 2: Add pricing fields to Product struct**

Modify the existing `Product` struct to add two fields after `Status`:

```go
type Product struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Price           Money     `json:"price"`
	Variants        []Variant `json:"variants,omitempty"`
	Status          string    `json:"status"`
	PricingModel    string    `json:"pricing_model,omitempty"`
	PricePerRequest float64   `json:"price_per_request,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
```

- [ ] **Step 3: Add PaymentAuth field to WellKnownResponse**

Add to the existing `WellKnownResponse` struct after `Currencies`:

```go
PaymentAuth *PaymentAuthConfig `json:"payment_auth,omitempty"`
```

- [ ] **Step 4: Verify it compiles**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/shared" && go build ./...`

- [ ] **Step 5: Commit**

```bash
git add shared/ace/types.go
git commit -m "feat: add payment auth and pricing types"
```

---

### Task 2: Payment Validator

**Files:**
- Create: `ace-server/internal/payment/validator.go`
- Create: `ace-server/internal/payment/validator_test.go`

- [ ] **Step 1: Write the failing test**

Create `ace-server/internal/payment/validator_test.go`:

```go
package payment

import (
	"context"
	"testing"
)

func TestMockValidator_AlwaysValid(t *testing.T) {
	v := NewMockValidator()
	result, err := v.Validate(context.Background(), "mock", "any_token", 1.50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected valid")
	}
	if result.TransactionID == "" {
		t.Fatal("expected transaction ID")
	}
}

func TestMockValidator_UnknownProvider(t *testing.T) {
	v := NewMockValidator()
	result, err := v.Validate(context.Background(), "stripe", "token", 1.50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid for unknown provider")
	}
}

func TestParsePaymentHeader_Valid(t *testing.T) {
	provider, token, ok := ParsePaymentHeader("mock:my_token_123")
	if !ok {
		t.Fatal("expected ok")
	}
	if provider != "mock" || token != "my_token_123" {
		t.Fatalf("got provider=%q token=%q", provider, token)
	}
}

func TestParsePaymentHeader_ColonInToken(t *testing.T) {
	provider, token, ok := ParsePaymentHeader("x402:0x123:456:789")
	if !ok {
		t.Fatal("expected ok")
	}
	if provider != "x402" || token != "0x123:456:789" {
		t.Fatalf("got provider=%q token=%q", provider, token)
	}
}

func TestParsePaymentHeader_Invalid(t *testing.T) {
	_, _, ok := ParsePaymentHeader("no_colon")
	if ok {
		t.Fatal("expected not ok")
	}
	_, _, ok = ParsePaymentHeader("")
	if ok {
		t.Fatal("expected not ok for empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-server" && go test ./internal/payment/ -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Write implementation**

Create `ace-server/internal/payment/validator.go`:

```go
package payment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// PaymentResult holds the result of a payment validation.
type PaymentResult struct {
	Valid            bool
	TransactionID    string
	BalanceRemaining *float64
}

// Validator validates payment tokens from different providers.
type Validator interface {
	Validate(ctx context.Context, provider, token string, price float64) (PaymentResult, error)
}

// MockValidator accepts any token for the "mock" provider.
type MockValidator struct{}

// NewMockValidator creates a new MockValidator.
func NewMockValidator() *MockValidator {
	return &MockValidator{}
}

// Validate checks a payment token. Mock provider always succeeds.
func (v *MockValidator) Validate(_ context.Context, provider, token string, price float64) (PaymentResult, error) {
	if provider != "mock" {
		return PaymentResult{Valid: false}, nil
	}
	b := make([]byte, 8)
	rand.Read(b)
	return PaymentResult{
		Valid:         true,
		TransactionID: "txn_mock_" + hex.EncodeToString(b),
	}, nil
}

// ParsePaymentHeader parses "provider:token" from the X-ACE-Payment header.
// Splits on the first colon only — tokens may contain colons.
func ParsePaymentHeader(header string) (provider, token string, ok bool) {
	if header == "" {
		return "", "", false
	}
	idx := strings.Index(header, ":")
	if idx <= 0 || idx == len(header)-1 {
		return "", "", false
	}
	return header[:idx], header[idx+1:], true
}
```

- [ ] **Step 4: Run tests**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-server" && go test ./internal/payment/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add ace-server/internal/payment/
git commit -m "feat: add PaymentValidator interface with mock provider"
```

---

### Task 3: Dual Mode Auth Middleware

**Files:**
- Modify: `ace-server/internal/middleware/auth.go`
- Create: `ace-server/internal/middleware/auth_test.go`

- [ ] **Step 1: Write the failing test**

Create `ace-server/internal/middleware/auth_test.go`:

```go
package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicroldan/ans/ace-server/internal/payment"
	"github.com/nicroldan/ans/ace-server/internal/store"
)

func TestDualAuth_APIKey(t *testing.T) {
	s := store.New()
	resp, key := s.CreateAPIKey("test-agent", []string{"catalog:read"})
	_ = resp

	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, true, []string{"mock"}, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-ACE-Key", key)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDualAuth_PaymentToken(t *testing.T) {
	s := store.New()
	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, true, []string{"mock"}, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-ACE-Payment", "mock:test_token_123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDualAuth_NoAuth_Returns402(t *testing.T) {
	s := store.New()
	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, true, []string{"mock"}, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d", w.Code)
	}
}

func TestDualAuth_NoAuth_PaymentDisabled_Returns401(t *testing.T) {
	s := store.New()
	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, false, nil, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDualAuth_InvalidPaymentToken(t *testing.T) {
	s := store.New()
	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, true, []string{"mock"}, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-ACE-Payment", "stripe:invalid_token") // mock validator rejects non-mock providers
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDualAuth_BothHeaders_APIKeyWins(t *testing.T) {
	s := store.New()
	_, key := s.CreateAPIKey("test-agent", []string{"catalog:read"})
	validator := payment.NewMockValidator()
	handler := DualAuth(s, validator, true, []string{"mock"}, "USD", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, _ := GetActor(r)
		if actor != "test-agent" {
			t.Fatalf("expected actor test-agent, got %s", actor)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-ACE-Key", key)
	req.Header.Set("X-ACE-Payment", "mock:token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-server" && go test ./internal/middleware/ -v`
Expected: FAIL — `DualAuth` not found

- [ ] **Step 3: Add DualAuth function to auth.go**

Add to `ace-server/internal/middleware/auth.go`:

```go
import (
	"github.com/nicroldan/ans/ace-server/internal/payment"
)

// DualAuth accepts either X-ACE-Key or X-ACE-Payment headers.
// If paymentEnabled is false, only API keys are accepted (401 on missing).
// If paymentEnabled is true and neither header is present, returns 402.
func DualAuth(s *store.MemoryStore, pv payment.Validator, paymentEnabled bool, providers []string, currency string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Priority 1: API Key
		if key := r.Header.Get("X-ACE-Key"); key != "" {
			apiKey, valid := s.ValidateAPIKey(key)
			if !valid {
				writeAuthError(w, http.StatusUnauthorized, "invalid_key", "Invalid API key")
				return
			}
			ctx := context.WithValue(r.Context(), ActorKey, apiKey.Name)
			ctx = context.WithValue(ctx, ActorTypeKey, "agent")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Priority 2: Payment Token
		if paymentHeader := r.Header.Get("X-ACE-Payment"); paymentHeader != "" {
			if !paymentEnabled {
				writeAuthError(w, http.StatusUnauthorized, "payment_not_supported", "This store does not accept payment auth")
				return
			}
			provider, token, ok := payment.ParsePaymentHeader(paymentHeader)
			if !ok {
				writeAuthError(w, http.StatusBadRequest, "invalid_payment", "Invalid X-ACE-Payment format, expected provider:token")
				return
			}
			result, err := pv.Validate(r.Context(), provider, token, 0)
			if err != nil {
				writeAuthError(w, http.StatusInternalServerError, "payment_error", "Payment validation failed")
				return
			}
			if !result.Valid {
				writeAuthError(w, http.StatusUnauthorized, "payment_rejected", "Payment token rejected")
				return
			}
			ctx := context.WithValue(r.Context(), ActorKey, "payment:"+result.TransactionID)
			ctx = context.WithValue(ctx, ActorTypeKey, "agent")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// No auth provided
		if paymentEnabled {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			resp := ace.PaymentRequiredResponse{
				Error: "Payment or API key required",
				Code:  "payment_required",
				Pricing: ace.PricingInfo{
					Price:             0,
					Currency:          currency,
					AcceptedProviders: providers,
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		writeAuthError(w, http.StatusUnauthorized, "auth_required", "X-ACE-Key header is required")
	})
}
```

- [ ] **Step 4: Run tests**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-server" && go test ./internal/middleware/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add ace-server/internal/middleware/
git commit -m "feat: add dual mode auth middleware (API key + payment)"
```

---

### Task 4: Pricing Headers Helper

**Files:**
- Modify: `ace-server/internal/handlers/helpers.go`

- [ ] **Step 1: Add WritePricingHeaders to helpers.go**

```go
import "fmt"

// WritePricingHeaders adds X-ACE-Price and X-ACE-Currency headers to the response.
func WritePricingHeaders(w http.ResponseWriter, price float64, balanceRemaining *float64) {
	w.Header().Set("X-ACE-Price", fmt.Sprintf("%.2f", price))
	w.Header().Set("X-ACE-Currency", "USD")
	if balanceRemaining != nil {
		w.Header().Set("X-ACE-Balance-Remaining", fmt.Sprintf("%.2f", *balanceRemaining))
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-server" && go build ./...`

- [ ] **Step 3: Commit**

```bash
git add ace-server/internal/handlers/helpers.go
git commit -m "feat: add WritePricingHeaders helper"
```

---

### Task 5: Update Buyer Handlers

**Files:**
- Modify: `ace-server/internal/handlers/buyer.go`

- [ ] **Step 1: Update Discovery to include payment_auth**

Modify the `Discovery` method to accept a `*ace.PaymentAuthConfig` field on the `BuyerHandler` struct and include it in the response:

Add field to BuyerHandler struct:
```go
paymentAuth *ace.PaymentAuthConfig
```

Update constructor to accept it:
```go
func NewBuyerHandler(s *store.MemoryStore, al *audit.Logger, storeID, name, baseURL string, paymentAuth *ace.PaymentAuthConfig) *BuyerHandler {
```

Update Discovery method to include `PaymentAuth: h.paymentAuth` in the response.

- [ ] **Step 2: Add Pricing endpoint**

Add to buyer.go:

```go
// Pricing handles GET /ace/v1/pricing (public, no auth required)
func (h *BuyerHandler) Pricing(w http.ResponseWriter, r *http.Request) {
	schedule := ace.PricingSchedule{
		DefaultCurrency: "USD",
		Endpoints: []ace.EndpointPrice{
			{Method: "GET", Path: "/ace/v1/products", Price: 0.00},
			{Method: "GET", Path: "/ace/v1/products/{id}", Price: 0.00},
			{Method: "POST", Path: "/ace/v1/cart", Price: 0.00},
			{Method: "POST", Path: "/ace/v1/cart/{id}/items", Price: 0.00},
			{Method: "POST", Path: "/ace/v1/orders", Price: 0.00},
			{Method: "POST", Path: "/ace/v1/orders/{id}/pay", Price: 0.00},
		},
	}
	writeJSON(w, http.StatusOK, schedule)
}
```

- [ ] **Step 3: Add pricing headers to ListProducts and GetProduct**

Before each `writeJSON` call in ListProducts and GetProduct, add:

```go
WritePricingHeaders(w, 0.00, nil)
```

Do the same for CreateCart, AddCartItem, GetCart, CreateOrder, GetOrder, Pay, PaymentStatus.

- [ ] **Step 4: Add cart restriction for per_request products**

In `AddCartItem`, after validating the product exists, add:

```go
if product.PricingModel == "per_request" {
    writeError(w, http.StatusBadRequest, "invalid_pricing_model", "Per-request products cannot be added to cart")
    return
}
```

- [ ] **Step 5: Verify it compiles**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-server" && go build ./...`

- [ ] **Step 6: Commit**

```bash
git add ace-server/internal/handlers/buyer.go
git commit -m "feat: add payment_auth to discovery, pricing endpoint, and pricing headers"
```

---

### Task 6: Wire Everything in main.go

**Files:**
- Modify: `ace-server/cmd/ace-server/main.go`

- [ ] **Step 1: Add env vars and payment validator initialization**

After existing env vars, add:

```go
paymentAuthEnabled := envOrDefault("PAYMENT_AUTH_ENABLED", "true") == "true"
paymentProviders := strings.Split(envOrDefault("PAYMENT_AUTH_PROVIDERS", "mock"), ",")
```

Add import for `"strings"` and `"github.com/nicroldan/ans/ace-server/internal/payment"`.

Create validator:
```go
paymentValidator := payment.NewMockValidator()
```

Create payment auth config:
```go
var paymentAuth *ace.PaymentAuthConfig
if paymentAuthEnabled {
    paymentAuth = &ace.PaymentAuthConfig{
        Enabled:         true,
        Header:          "X-ACE-Payment",
        Providers:       paymentProviders,
        DefaultCurrency: "USD",
    }
}
```

Pass `paymentAuth` to `NewBuyerHandler`.

- [ ] **Step 2: Replace aceAuth with dualAuth**

Replace:
```go
aceAuth := func(handler http.HandlerFunc) http.Handler {
    return middleware.ACEAuth(memStore, http.HandlerFunc(handler))
}
```

With:
```go
dualAuth := func(handler http.HandlerFunc) http.Handler {
    return middleware.DualAuth(memStore, paymentValidator, paymentAuthEnabled, paymentProviders, "USD", http.HandlerFunc(handler))
}
```

Update all buyer route registrations to use `dualAuth` instead of `aceAuth`.

- [ ] **Step 3: Add pricing route (public, no auth)**

After the discovery route:
```go
mux.HandleFunc("GET /ace/v1/pricing", buyerHandler.Pricing)
```

- [ ] **Step 4: Add startup log for payment auth**

```go
if paymentAuthEnabled {
    log.Printf("Payment auth: ENABLED (providers: %s)", strings.Join(paymentProviders, ", "))
} else {
    log.Printf("Payment auth: DISABLED (API key only)")
}
```

- [ ] **Step 5: Verify it compiles and run**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-server" && go build ./...`

- [ ] **Step 6: Commit**

```bash
git add ace-server/cmd/ace-server/main.go
git commit -m "feat: wire payment auth, pricing endpoint, and dual auth middleware"
```

---

### Task 7: End-to-End Manual Test

- [ ] **Step 1: Start ace-server**

```bash
cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-server" && go run ./cmd/ace-server
```

- [ ] **Step 2: Test discovery includes payment_auth**

```bash
curl -s http://localhost:8081/.well-known/agent-commerce | python3 -m json.tool
```

Expected: response includes `"payment_auth": {"enabled": true, ...}`

- [ ] **Step 3: Test 402 without auth**

```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/ace/v1/products
```

Expected: `402`

- [ ] **Step 4: Test payment-as-auth with mock**

```bash
curl -s http://localhost:8081/ace/v1/products -H "X-ACE-Payment: mock:test123"
```

Expected: 200 with products list and `X-ACE-Price` headers

- [ ] **Step 5: Test pricing endpoint (no auth)**

```bash
curl -s http://localhost:8081/ace/v1/pricing | python3 -m json.tool
```

Expected: pricing schedule with all endpoints

- [ ] **Step 6: Test API key still works**

```bash
curl -s http://localhost:8081/ace/v1/products -H "X-ACE-Key: <DEMO_KEY>"
```

Expected: 200 with products list

- [ ] **Step 7: Commit any fixes**

---

### Task 8: Update README and GETTING_STARTED

**Files:**
- Modify: `README.md`
- Modify: `GETTING_STARTED.md`

- [ ] **Step 1: Update README.md**

Add a "Headless Merchant" section after the existing architecture section explaining:
- Payment-as-auth: agents can pay per request without accounts
- Dual mode: API keys and payment tokens both work
- Per-request pricing model for API-style services
- Pricing headers on all responses
- Link to the spec doc for details

Update the architecture diagram to show the payment-as-auth flow.

- [ ] **Step 2: Update GETTING_STARTED.md**

Add examples showing:
- How to use payment-as-auth with mock provider
- The 402 response and what it means
- How to check pricing before paying
- How existing API key flow still works

- [ ] **Step 3: Commit**

```bash
git add README.md GETTING_STARTED.md
git commit -m "docs: add headless merchant features to README and getting started"
```

- [ ] **Step 4: Push to remote**

```bash
gh auth switch --user nicoroldan1
git push origin main
gh auth switch --user nicroldan_meli
```
