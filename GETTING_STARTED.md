# Getting Started with ACE

This guide walks you through running ACE locally, making your first purchase with a buyer agent, and connecting an AI agent (Claude, GPT, or any LLM) to an ACE-compatible store.

---

## Prerequisites

- **Go 1.22+** — [Install Go](https://go.dev/dl/)
- **curl** — for manual API testing (pre-installed on macOS/Linux)
- **git** — to clone the repo

Check your Go version:

```bash
go version
# go version go1.22.0 darwin/arm64 (or higher)
```

---

## 1. Clone and Build

```bash
git clone https://github.com/nicoroldan1/agent-commerce-protocol.git
cd agent-commerce-protocol
```

The repo is a Go workspace. No dependency installation needed — everything uses the standard library.

---

## 2. Start the Registry

The registry is the discovery index — agents query it to find stores.

```bash
# Terminal 1
cd registry
go run ./cmd/registry
```

```
Registry server listening on :8080
```

**Default port:** `8080`
Override with: `PORT=9000 go run ./cmd/registry`

---

## 3. Start a Demo Store (ACE Reference Server)

```bash
# Terminal 2
cd ace-server
go run ./cmd/ace-server
```

```
ACE Reference Server starting on :8081
Store: ACE Demo Store (store_demo_001)
Admin token: a1b2c3d4...  (auto-generated, copy from terminal)
Demo API key: e5f6a7b8...  (auto-generated, copy from terminal)
Discovery: http://localhost:8081/.well-known/agent-commerce
```

**Default port:** `8081`

The server starts with 7 pre-seeded products (headphones, coffee, keyboard, shoes, books, yoga mat, water bottle) and a ready-to-use API key.

**Important:** Both the admin token and demo API key are **generated randomly at each startup** and printed to the terminal. Copy them from the output to use in your requests. You can also set them explicitly via environment variables for reproducible setups.

**Environment variables (all optional):**

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8081` | Server port |
| `STORE_ID` | `store_demo_001` | Store identifier |
| `STORE_NAME` | `ACE Demo Store` | Store display name |
| `ADMIN_TOKEN` | *(auto-generated)* | Admin API bearer token |
| `DEMO_API_KEY` | *(auto-generated)* | Pre-seeded buyer API key |
| `BASE_URL` | `http://localhost:8081` | Public base URL |
| `PAYMENT_AUTH_ENABLED` | `true` | Enable payment-as-auth (agents can pay per request without API key) |
| `PAYMENT_AUTH_PROVIDERS` | `mock` | Comma-separated list of accepted payment providers |

---

## 4. Register the Store in the Registry

```bash
curl -X POST http://localhost:8080/registry/v1/stores \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ACE Demo Store",
    "well_known_url": "http://localhost:8081/.well-known/agent-commerce",
    "categories": ["electronics", "books", "sports"],
    "country": "US",
    "currencies": ["USD"]
  }'
```

---

## 5. Verify Everything Works

**Discover the store:**
```bash
curl http://localhost:8081/.well-known/agent-commerce | jq .
```

```json
{
  "store_id": "store_demo_001",
  "name": "ACE Demo Store",
  "version": "ace/0.1",
  "ace_base_url": "http://localhost:8081/ace/v1",
  "capabilities": ["catalog", "cart", "orders", "payments"],
  "currencies": ["USD"],
  "payment_protocols": ["x402", "mpp", "stripe"],
  "auth": { "type": "api_key", "header": "X-ACE-Key" }
}
```

**Browse the catalog (with API key):**
```bash
curl http://localhost:8081/ace/v1/products \
  -H "X-ACE-Key: <YOUR_DEMO_API_KEY>" | jq .
```

**Browse the catalog (with payment-as-auth — no API key needed):**
```bash
curl http://localhost:8081/ace/v1/products \
  -H "X-ACE-Payment: mock:any_token_here" | jq .
```

**See what happens with no auth:**
```bash
curl -i http://localhost:8081/ace/v1/products
# HTTP 402 Payment Required
# {"error":"Payment or API key required","code":"payment_required","pricing":{"price":0,"currency":"USD","accepted_providers":["mock"]}}
```

**Check pricing before paying:**
```bash
curl http://localhost:8081/ace/v1/pricing | jq .
```

All responses include pricing headers:
```
X-ACE-Price: 0.00
X-ACE-Currency: USD
```

---

## 6. Run the Demo Buyer Agent

The demo agent is a Go CLI that simulates an autonomous buyer: discovers stores, browses the catalog, creates a cart, places an order, and initiates payment.

```bash
# Terminal 3
cd agent-buyer

# Option A: Agent discovers stores via the registry
go run ./cmd/buyer --registry http://localhost:8080 --key <YOUR_DEMO_API_KEY>

# Option B: Agent connects directly to a store (skip registry)
go run ./cmd/buyer --store http://localhost:8081/.well-known/agent-commerce --key <YOUR_DEMO_API_KEY>
```

Expected output:

```
=== ANS Demo Agent Buyer ===

Step 1: Discovering stores...
  Querying registry at http://localhost:8080...
  Found 1 store(s):
    - "ACE Demo Store" (store_...) - healthy
  Selecting first store...

Step 2: Connecting to store...
  Store: "ACE Demo Store"
  Version: ace/0.1
  Capabilities: catalog, cart, orders, payments

Step 3: Browsing catalog...
  Found 7 product(s):
    1. Wireless Headphones - $79.99 (prod_...)
    2. Organic Coffee Beans - $24.99 (prod_...)
    ...
  Selecting first 2 product(s)...

Step 4: Creating cart...
  Cart created: cart_...
  Adding "Wireless Headphones" x1...
  Adding "Organic Coffee Beans" x2...
  Cart total: $129.97

Step 5: Placing order...
  Order created: order_...
  Status: pending
  Items: 2 item(s), total $129.97

Step 6: Initiating payment...
  Payment initiated: pay_...
  Provider: stripe
  Payment URL: https://checkout.stripe.com/...

Step 7: Checking payment status...
  Payment status: pending
  Order status: pending

=== Demo complete! Full purchase flow successful. ===
```

---

## 7. Full Manual Walkthrough (curl)

This is what an agent does, step by step:

### Step 1 — Discover

```bash
curl http://localhost:8081/.well-known/agent-commerce
```

### Step 2 — Browse catalog

```bash
curl "http://localhost:8081/ace/v1/products?q=keyboard" \
  -H "X-ACE-Key: <YOUR_DEMO_API_KEY>"
```

### Step 3 — Create cart

```bash
curl -X POST http://localhost:8081/ace/v1/cart \
  -H "X-ACE-Key: <YOUR_DEMO_API_KEY>" \
  -H "Content-Type: application/json"
# → { "id": "cart_abc123", ... }
```

### Step 4 — Add item

```bash
curl -X POST http://localhost:8081/ace/v1/cart/cart_abc123/items \
  -H "X-ACE-Key: <YOUR_DEMO_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{ "product_id": "PRODUCT_ID_HERE", "quantity": 1 }'
```

### Step 5 — Place order

```bash
curl -X POST http://localhost:8081/ace/v1/orders \
  -H "X-ACE-Key: <YOUR_DEMO_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{ "cart_id": "cart_abc123" }'
# → { "id": "order_def456", "status": "pending", "total": { "amount": 14999, "currency": "USD" } }
```

### Step 6 — Initiate payment

```bash
curl -X POST http://localhost:8081/ace/v1/orders/order_def456/pay \
  -H "X-ACE-Key: <YOUR_DEMO_API_KEY>" \
  -H "Content-Type: application/json" \
  -d '{ "protocol": "stripe" }'
```

### Step 7 — Check payment status

```bash
curl http://localhost:8081/ace/v1/orders/order_def456/pay/status \
  -H "X-ACE-Key: <YOUR_DEMO_API_KEY>"
```

---

## 8. Using ACE with an AI Agent (Claude, GPT, or any LLM)

Any AI agent that can make HTTP calls can use ACE. The protocol is designed to be self-describing — an LLM can read `/.well-known/agent-commerce` and understand what to do next.

### With Claude (via tool use)

Give Claude the following tool definitions and it can autonomously complete purchases:

```json
[
  {
    "name": "discover_store",
    "description": "Discover an ACE-compatible store and its capabilities",
    "input_schema": {
      "type": "object",
      "properties": {
        "well_known_url": { "type": "string", "description": "URL of the store's .well-known/agent-commerce endpoint" }
      },
      "required": ["well_known_url"]
    }
  },
  {
    "name": "search_products",
    "description": "Search the store catalog",
    "input_schema": {
      "type": "object",
      "properties": {
        "query": { "type": "string" },
        "limit": { "type": "integer", "default": 20 }
      }
    }
  },
  {
    "name": "create_cart",
    "description": "Create a new shopping cart"
  },
  {
    "name": "add_to_cart",
    "description": "Add a product to the cart",
    "input_schema": {
      "type": "object",
      "properties": {
        "cart_id": { "type": "string" },
        "product_id": { "type": "string" },
        "quantity": { "type": "integer" }
      },
      "required": ["cart_id", "product_id", "quantity"]
    }
  },
  {
    "name": "place_order",
    "description": "Convert cart to order",
    "input_schema": {
      "type": "object",
      "properties": {
        "cart_id": { "type": "string" }
      },
      "required": ["cart_id"]
    }
  },
  {
    "name": "pay",
    "description": "Initiate payment for an order",
    "input_schema": {
      "type": "object",
      "properties": {
        "order_id": { "type": "string" },
        "protocol": { "type": "string", "enum": ["x402", "mpp", "stripe"] }
      },
      "required": ["order_id", "protocol"]
    }
  }
]
```

**System prompt:**

```
You are a buyer agent. You can discover ACE-compatible stores and make purchases autonomously.

The store URL is: http://localhost:8081/.well-known/agent-commerce
Your API key is: <YOUR_DEMO_API_KEY>

When asked to buy something:
1. Call discover_store to understand the store's capabilities
2. Use search_products to find the right item
3. Create a cart, add the item, and place an order
4. Initiate payment using the protocol declared in the store's payment_protocols field
```

**Example prompt to Claude:**

```
Buy me wireless headphones. Budget: $100. Use stripe for payment.
```

Claude will autonomously execute all 7 steps.

### With GPT (via function calling)

Same approach — use the same tool definitions as OpenAI function schemas. GPT-4 and later models handle the ACE flow without any custom training.

### With any LLM (zero-shot)

Paste the following into any LLM that supports tool/function calling:

```
You have access to an ACE-compatible store.

Base URL: http://localhost:8081/ace/v1
API Key header: X-ACE-Key: <YOUR_DEMO_API_KEY>
Discovery: GET http://localhost:8081/.well-known/agent-commerce

ACE protocol endpoints:
- GET  /ace/v1/products?q={query}
- POST /ace/v1/cart
- POST /ace/v1/cart/{id}/items  body: { "product_id": "...", "quantity": N }
- POST /ace/v1/orders           body: { "cart_id": "..." }
- POST /ace/v1/orders/{id}/pay  body: { "protocol": "stripe" }

Task: Buy a coffee product for me.
```

---

## 9. Admin Operations

Manage your store with the Admin API (requires `Authorization: Bearer <YOUR_ADMIN_TOKEN>`):

**Create a product:**
```bash
curl -X POST http://localhost:8081/api/v1/stores/store_demo_001/products \
  -H "Authorization: Bearer <YOUR_ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Product",
    "description": "A great product",
    "price": { "amount": 4999, "currency": "USD" }
  }'
```

**Issue an API key to a buyer agent:**
```bash
curl -X POST http://localhost:8081/api/v1/stores/store_demo_001/api-keys \
  -H "Authorization: Bearer <YOUR_ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "agent_name": "my-buyer-agent",
    "scopes": ["catalog:read", "cart:write", "orders:write", "payments:write"]
  }'
```

**View audit trail:**
```bash
curl http://localhost:8081/api/v1/stores/store_demo_001/audit-logs \
  -H "Authorization: Bearer <YOUR_ADMIN_TOKEN>" | jq .
```

**Configure policies:**
```bash
curl -X PUT http://localhost:8081/api/v1/stores/store_demo_001/policies \
  -H "Authorization: Bearer <YOUR_ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '[
    { "action": "order.refund", "effect": "approval" },
    { "action": "product.publish", "effect": "allow" }
  ]'
```

---

## 10. Deploy Your Own Store

To make your ACE store publicly accessible (so real agents can find it):

### Option A — Deploy to Railway/Render/Fly.io

```bash
# Set environment variables in your hosting provider:
PORT=8081
STORE_ID=my-store-001
STORE_NAME=My Store
ADMIN_TOKEN=<strong-secret>
BASE_URL=https://my-store.example.com

# Build and deploy
go build -o ace-server ./cmd/ace-server
```

### Option B — Docker

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ace-server ./cmd/ace-server

FROM alpine:latest
COPY --from=builder /app/ace-server /ace-server
EXPOSE 8081
CMD ["/ace-server"]
```

### Register in the public registry

Once your store is live, register it:

```bash
curl -X POST https://registry.ace-protocol.dev/registry/v1/stores \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Store",
    "well_known_url": "https://my-store.example.com/.well-known/agent-commerce",
    "categories": ["electronics"],
    "country": "US",
    "currencies": ["USD"]
  }'
```

Any agent that queries the public registry will now be able to find and transact with your store.

---

## Troubleshooting

| Problem | Solution |
|---------|---------|
| `connection refused :8080` | Registry is not running. Start it with `cd registry && go run ./cmd/registry` |
| `connection refused :8081` | ACE server is not running. Start it with `cd ace-server && go run ./cmd/ace-server` |
| `401 Unauthorized` | Check you're sending `X-ACE-Key: <YOUR_DEMO_API_KEY>` |
| `No stores found in registry` | Register the store first (step 4 above) |
| `go: module not found` | Run from the repo root (where `go.work` is), not from a subdirectory |
| Port already in use | Change port: `PORT=9090 go run ./cmd/ace-server` |

---

## What's Next

- Read the [full protocol spec](ace-spec/README.md)
- Check the [architecture and business docs](00-negocio.md)
- Open an issue or PR to contribute
- Deploy your own store and register it in the network

The more stores that implement ACE, the more useful buyer agents become — and the closer we get to a truly open agentic commerce ecosystem.
