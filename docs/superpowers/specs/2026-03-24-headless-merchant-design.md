# Headless Merchant Features for ACE Protocol

**Date:** 2026-03-24
**Status:** Approved

## Problem

The ACE protocol currently requires API keys for all buyer interactions. The emerging "headless merchant" model (as described in Noah Levine's "Entering the Era of the Headless Merchant") enables agents to pay per request without accounts or API keys. ACE needs to support this model while keeping existing API key auth working.

## Decision

Implement three features that align ACE with the headless merchant vision:
1. **Payment-as-auth (dual mode)** — endpoints accept API keys OR payment tokens
2. **Per-request pricing** — products can declare a per-call price model
3. **Pricing headers** — all buyer API responses include cost information

## Money Representation

The existing `Money` struct uses `int64` cents. Per-request pricing needs sub-cent precision (e.g., $0.003). Rather than changing the existing `Money` type (which would break the cart/order flow), we introduce a separate `float64` representation for per-request pricing only:

- **Existing `Money` struct** (unchanged): used for product prices, cart totals, order totals. Integer cents.
- **New `price_per_request` field**: `float64` in USD. Used only for per-request pricing and headers.
- **`X-ACE-Price` header**: string-formatted float in USD (e.g., `"0.003"`).
- **402 response `pricing.price`**: `float64` in USD.

This avoids changing the existing type system while supporting sub-cent pricing for the headless merchant use case.

## 1. Payment-as-Auth (Dual Mode)

### Auth Modes

Buyer API endpoints accept two authentication modes. The store declares which modes it supports in `.well-known/agent-commerce`.

**Mode 1 — API Key (existing, unchanged):**
```
GET /ace/v1/products
X-ACE-Key: ak_abc123
```

**Mode 2 — Payment token (new):**
```
GET /ace/v1/products
X-ACE-Payment: <provider>:<receipt_token>
```

Examples:
- `stripe:pi_abc123` — Stripe payment intent
- `x402:0xabc123...` — x402 on-chain receipt
- `mock:anything` — Mock provider for testing (always valid)

**Parsing rule:** split on the FIRST colon only. Everything after the first `:` is the token (tokens may contain colons).

### 402 Payment Required

When a request includes neither API key nor payment token, the server returns HTTP 402:

```json
{
  "error": "Payment or API key required",
  "code": "payment_required",
  "pricing": {
    "price": 0.50,
    "currency": "USD",
    "accepted_providers": ["stripe", "x402", "mock"],
    "details_url": "https://store.example.com/ace/v1/pricing"
  }
}
```

This uses a dedicated `PaymentRequiredResponse` type (not `ErrorResponse`) since it includes the nested `pricing` object:

```go
type PaymentRequiredResponse struct {
    Error   string      `json:"error"`
    Code    string      `json:"code"`
    Pricing PricingInfo `json:"pricing"`
}

type PricingInfo struct {
    Price             float64  `json:"price"`
    Currency          string   `json:"currency"`
    AcceptedProviders []string `json:"accepted_providers"`
    DetailsURL        string   `json:"details_url,omitempty"`
}
```

### Well-Known Extension

The `.well-known/agent-commerce` response adds a `payment_auth` section:

```json
{
  "store_id": "store_abc123",
  "name": "Acme Widget Store",
  "version": "1.0.0",
  "ace_base_url": "https://acme.example.com/ace/v1",
  "capabilities": ["catalog", "cart", "orders", "payments"],
  "auth": {
    "type": "api_key",
    "header": "X-ACE-Key"
  },
  "payment_auth": {
    "enabled": true,
    "header": "X-ACE-Payment",
    "providers": ["stripe", "x402", "mock"],
    "default_currency": "USD"
  },
  "currencies": ["USD"]
}
```

If `payment_auth.enabled` is `false` or omitted, the store only accepts API keys (backwards compatible).

The `WellKnownResponse` struct gains a new field:

```go
type PaymentAuthConfig struct {
    Enabled         bool     `json:"enabled"`
    Header          string   `json:"header"`
    Providers       []string `json:"providers"`
    DefaultCurrency string   `json:"default_currency"`
}
```

Added to `WellKnownResponse` as `PaymentAuth *PaymentAuthConfig \`json:"payment_auth,omitempty"\``.

### Configuration

Payment auth is configured via environment variables on the ace-server:
- `PAYMENT_AUTH_ENABLED` — `true` or `false` (default: `true`)
- `PAYMENT_AUTH_PROVIDERS` — comma-separated list (default: `mock`)

When enabled, the server includes `payment_auth` in the `.well-known` response and accepts `X-ACE-Payment` headers.

### Payment Validation

ACE defines the **interface** for presenting payment in an HTTP request. ACE does NOT process payments. Each store validates tokens against the provider it supports.

The validation flow:
1. Middleware extracts `X-ACE-Payment` header
2. Parses `provider:token` format
3. Delegates to a `PaymentValidator` interface
4. The validator checks with the external provider (or auto-approves for `mock`)
5. If valid, request proceeds. If invalid, returns 401.

```go
type PaymentValidator interface {
    Validate(ctx context.Context, provider, token string, price float64) (PaymentResult, error)
}

type PaymentResult struct {
    Valid           bool
    TransactionID   string
    BalanceRemaining *float64 // nil if not applicable
}
```

The `mock` provider always returns `Valid: true` with a generated transaction ID. Mock tokens are intentionally reusable (no replay protection). Real providers SHOULD enforce single-use tokens, but ACE does not mandate this — it is the provider's responsibility.

### Auth Priority

When both `X-ACE-Key` and `X-ACE-Payment` are present, API key takes priority (payment header is ignored). This prevents double-charging.

## 2. Per-Request Pricing

### Product Pricing Model

Products gain two optional fields:

```json
{
  "id": "prod_1",
  "name": "Image Generation API",
  "price": { "amount": 0, "currency": "USD" },
  "pricing_model": "per_request",
  "price_per_request": 0.003
}
```

- `pricing_model`: `"fixed"` (default) or `"per_request"`
- `price_per_request`: cost in USD per API call (decimal, e.g., `0.003` = $0.003)

For `fixed` products, the existing cart → order → pay flow is used. For `per_request` products, the agent pays on each request via `X-ACE-Payment` — no cart or order needed.

**Cart restriction:** Adding a `per_request` product to a cart returns `400 Bad Request` with error code `"invalid_pricing_model"`. Per-request products cannot be purchased through the cart/order flow.

### Pricing Endpoint

A new **public** endpoint (no auth required) exposes the store's pricing schedule so agents can discover costs before paying:

```
GET /ace/v1/pricing
```

Response:
```json
{
  "default_currency": "USD",
  "endpoints": [
    { "method": "GET", "path": "/ace/v1/products", "price": 0.00 },
    { "method": "GET", "path": "/ace/v1/products/{id}", "price": 0.00 },
    { "method": "POST", "path": "/ace/v1/cart", "price": 0.00 },
    { "method": "POST", "path": "/ace/v1/orders", "price": 0.00 },
    { "method": "POST", "path": "/ace/v1/orders/{id}/pay", "price": 0.00 }
  ]
}
```

```go
type PricingSchedule struct {
    DefaultCurrency string           `json:"default_currency"`
    Endpoints       []EndpointPrice  `json:"endpoints"`
}

type EndpointPrice struct {
    Method string  `json:"method"`
    Path   string  `json:"path"`
    Price  float64 `json:"price"`
}
```

By default, all existing endpoints are free (price 0.00). Per-request product pricing is on the product itself in the catalog (`price_per_request` field) — not duplicated here.

## 3. Pricing Headers in Responses

All buyer API responses include pricing headers:

```
HTTP/1.1 200 OK
Content-Type: application/json
X-ACE-Price: 0.003
X-ACE-Currency: USD
```

If the agent used a payment token with balance tracking:
```
X-ACE-Balance-Remaining: 4.85
```

If the endpoint is free:
```
X-ACE-Price: 0.00
X-ACE-Currency: USD
```

Headers are added by a helper function `WritePricingHeaders(w, price, balanceRemaining)` called by each buyer handler before writing the response. This is simpler than a response-intercepting middleware and consistent with the existing codebase style where handlers call helpers directly.

## Implementation Scope

### Files Changed

| Component | Change |
|-----------|--------|
| `shared/ace/types.go` | Add PaymentAuthConfig, PaymentRequiredResponse, PricingInfo, PricingSchedule, EndpointPrice, pricing fields on Product |
| `ace-server/internal/middleware/auth.go` | Dual mode: accept API key OR payment token; return 402 when payment_auth enabled |
| `ace-server/internal/payment/validator.go` | PaymentValidator interface + MockValidator |
| `ace-server/internal/payment/validator_test.go` | Unit tests |
| `ace-server/internal/handlers/helpers.go` | Add WritePricingHeaders helper function |
| `ace-server/internal/handlers/buyer.go` | Update Discovery to include payment_auth; add pricing endpoint; call WritePricingHeaders |
| `ace-server/cmd/ace-server/main.go` | Wire payment validator, read PAYMENT_AUTH env vars, new routes |

### What Does NOT Change

- The purchase flow (cart → order → pay) for `fixed` pricing products
- The Admin API (admin always uses admin token)
- The registry and search system
- Existing API keys continue to work
- The registry token auth for product sync

## Auth Decision Matrix

| X-ACE-Key | X-ACE-Payment | Store supports payment_auth | Result |
|-----------|---------------|----------------------------|--------|
| Present | Absent | Any | Validate API key (existing flow) |
| Present | Present | Any | Validate API key (payment ignored) |
| Absent | Present | Yes | Validate payment token |
| Absent | Present | No | 401 Unauthorized |
| Absent | Absent | Yes | 402 Payment Required |
| Absent | Absent | No | 401 Unauthorized |

## Future Extensions (Not in Scope)

- Real Stripe payment validation (requires Stripe API keys)
- Real x402 on-chain validation (requires blockchain integration)
- Per-endpoint configurable pricing (all endpoints free for now except per_request products)
- Balance/wallet management
- Settlement and payouts to merchants
