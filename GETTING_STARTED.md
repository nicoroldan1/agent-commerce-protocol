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

## 11. Connect an Existing Store (ACE Connect)

ACE Connect turns any existing e-commerce store into an ACE-compatible endpoint with a single command. It fetches your catalog, serves ACE protocol endpoints, registers in the registry, and syncs products periodically.

Currently supported: **Shopify**. Planned: Tiendanube, WooCommerce, CSV import.

### Shopify Setup

Before running ACE Connect, create a Custom App in Shopify to get an API access token:

1. Go to your Shopify Admin panel.
2. Navigate to **Settings > Apps and sales channels > Develop apps**.
3. Click **Create an app** and name it (e.g. "ACE Connect").
4. Under **Configuration**, click **Configure Admin API scopes**.
5. Enable the `read_products` scope (and `read_orders` if you want order sync later).
6. Click **Save**, then go to the **API credentials** tab.
7. Click **Install app** and confirm.
8. Copy the **Admin API access token** -- this is your `--token` value.

### Run ACE Connect

```bash
npx ace-connect shopify \
  --shop mystore.myshopify.com \
  --token shp_your_access_token \
  --registry http://localhost:8080 \
  --port 8082
```

Expected output:

```
[ace-connect] Connecting to Shopify: mystore.myshopify.com
[ace-connect] Currency: USD
[ace-connect] Loaded 47 products
[ace-connect] ACE server running on :8082
[ace-connect] Registered in registry as str_abc123
[sync] Synced 47 products to registry
```

ACE Connect embeds a full ACE server, so agents interact with it exactly like the reference server. It exposes `/.well-known/agent-commerce`, `/ace/v1/products`, `/ace/v1/cart`, and `/ace/v1/orders`.

### CLI Options

| Option | Default | Description |
|--------|---------|-------------|
| `--shop <domain>` | _(required)_ | Your Shopify `.myshopify.com` domain |
| `--token <token>` | _(required)_ | Shopify Admin API access token |
| `--registry <url>` | _(none)_ | ACE Registry URL for discovery and cross-store search |
| `--port <port>` | `8081` | Port for the embedded ACE HTTP server |
| `--sync-interval <sec>` | `300` | Re-fetch interval in seconds |
| `--country <code>` | `US` | Country code for registry filtering |
| `--categories <list>` | _(none)_ | Comma-separated categories for registry classification |
| `--currency <code>` | _(auto)_ | Override currency (auto-detected from Shopify if omitted) |

### Verify the Connected Store

Once running, the connected store works like any other ACE store:

```bash
# Discovery
curl http://localhost:8082/.well-known/agent-commerce | jq .

# Browse catalog (payment-as-auth, no API key needed)
curl http://localhost:8082/ace/v1/products \
  -H "X-ACE-Payment: mock:agent-123" | jq .

# Search from the registry (products were synced automatically)
curl "http://localhost:8080/registry/v1/search?q=shoes&country=US" | jq .
```

---

## 12. Use the MCP Buyer with Claude

The ACE Buyer MCP server gives Claude (or any MCP-compatible AI) direct access to the ACE protocol. Claude can discover stores, browse catalogs, manage carts, place orders, and pay -- all through natural conversation.

### Claude Desktop Configuration

Add this to your Claude Desktop config file (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "ace-buyer": {
      "command": "node",
      "args": ["/absolute/path/to/mcp-buyer/dist/index.js"],
      "env": {
        "ACE_REGISTRY_URL": "http://localhost:8080",
        "ACE_STORE_URL": "http://localhost:8081",
        "ACE_PAYMENT_PROVIDER": "mock"
      }
    }
  }
}
```

Replace `/absolute/path/to/mcp-buyer/dist/index.js` with the actual path on your machine. Build first if you have not already:

```bash
cd mcp-buyer
npm install
npm run build
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ACE_REGISTRY_URL` | No | -- | Registry URL for multi-store discovery |
| `ACE_STORE_URL` | No | -- | Single store URL (direct connection) |
| `ACE_API_KEY` | No | -- | API key for store authentication |
| `ACE_PAYMENT_PROVIDER` | No | `mock` | Payment provider for orders |
| `ACE_PAYMENT_TOKEN` | No | -- | Token for the payment provider |

At least one of `ACE_REGISTRY_URL` or `ACE_STORE_URL` must be set.

### Two Modes

**Single-Store Mode** -- set only `ACE_STORE_URL`. Claude connects directly to one store. Discovery tools are not available.

**Registry Mode** -- set `ACE_REGISTRY_URL` (and optionally `ACE_STORE_URL` as a default). All 13 tools are available. Claude can discover stores, search across them, and purchase from any of them.

### Available Tools (13 total)

**Discovery (Registry Mode only)**

| Tool | Description |
|------|-------------|
| `discover_stores` | Search for ACE stores by name, category, or country |
| `search_products` | Search products across all registered stores |

**Catalog**

| Tool | Description |
|------|-------------|
| `browse_store` | List products from a store with optional filters |
| `get_product` | Get full product details, variants, and pricing |
| `get_pricing` | Get a store's pricing schedule for all endpoints |

**Purchase**

| Tool | Description |
|------|-------------|
| `create_cart` | Create a new shopping cart |
| `get_cart` | Get the current state of a cart |
| `add_to_cart` | Add a product to a cart |
| `shipping_quote` | Get shipping options and prices |
| `place_order` | Convert a cart into an order |
| `pay_order` | Pay for an order using the configured provider |
| `get_order` | Get full order details and payment status |
| `payment_status` | Check payment status for an order |

### Example Conversation

```
User: Find me a mechanical keyboard under $100.

Claude: I'll search for mechanical keyboards in that price range.
        [calls search_products with query="mechanical keyboard", price_max=100]

        I found 3 options:
        1. KeyTech MK-75 - $79.99
        2. ClickBoard Pro - $89.50
        3. TypeMaster 60% - $64.00

User: I'll take the TypeMaster.

Claude: Great choice. Let me set up your order.
        [calls create_cart]
        [calls add_to_cart with product_id="typemaster-60", quantity=1]
        [calls place_order with the cart_id]

        Your order ORD-1234 is ready. Total: $64.00.
        Shall I proceed with payment?

User: Yes, go ahead.

Claude: [calls pay_order with order_id="ORD-1234"]
        [calls payment_status with order_id="ORD-1234"]

        Payment confirmed. Order ORD-1234 is paid and being processed.
```

---

## 13. Scalable Search with Elasticsearch

By default, the registry uses in-memory storage. For production-like search across thousands of products from multiple stores, enable Elasticsearch.

### Option A: Docker Compose (recommended)

The included `docker-compose.yml` starts Elasticsearch, the registry, and the demo store together:

```bash
docker-compose up -d
```

This starts:
- **Elasticsearch** on `localhost:9200` (single-node, security disabled for local dev)
- **Registry** on `localhost:8080` (connected to Elasticsearch automatically)
- **ACE Demo Store** on `localhost:8081`

### Option B: Manual Setup

Start Elasticsearch separately (Docker or native), then pass the URL to the registry:

```bash
# Terminal 1: Start Elasticsearch
docker run -d --name es \
  -e discovery.type=single-node \
  -e xpack.security.enabled=false \
  -e "ES_JAVA_OPTS=-Xms512m -Xmx512m" \
  -p 9200:9200 \
  docker.elastic.co/elasticsearch/elasticsearch:8.17.0

# Terminal 2: Start registry with Elasticsearch
cd registry
ELASTICSEARCH_URL=http://localhost:9200 go run ./cmd/registry
```

If `ELASTICSEARCH_URL` is not set, the registry falls back to in-memory search (suitable for development but does not support full-text search across synced products).

### Sync Products to the Search Index

When a store registers, the registry returns a `registry_token`. Use this token to push products into the search index:

```bash
# Step 1: Register a store and capture the token
curl -s -X POST http://localhost:8080/registry/v1/stores \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ACE Demo Store",
    "well_known_url": "http://localhost:8081/.well-known/agent-commerce",
    "categories": ["electronics", "books", "sports"],
    "country": "US"
  }' | jq .

# Response includes:
# {
#   "id": "str_abc123",
#   "registry_token": "rtk_xxxxxxxx",
#   ...
# }
```

```bash
# Step 2: Sync products using the registry token
curl -X POST http://localhost:8080/registry/v1/products/sync \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer rtk_xxxxxxxx" \
  -d '{
    "products": [
      {
        "id": "prod_001",
        "name": "Wireless Keyboard",
        "description": "Compact mechanical keyboard with Bluetooth",
        "price": { "amount": 7999, "currency": "USD" },
        "category": "electronics",
        "in_stock": true
      },
      {
        "id": "prod_002",
        "name": "Yoga Mat",
        "description": "Non-slip exercise mat, 6mm thick",
        "price": { "amount": 2999, "currency": "USD" },
        "category": "sports",
        "in_stock": true
      }
    ]
  }'
```

Note: ACE Connect syncs products automatically when `--registry` is set, so manual sync is only needed for the reference server or custom integrations.

### Search Across All Stores

Once products are synced, agents (or you) can search across every registered store:

```bash
# Full-text search
curl "http://localhost:8080/registry/v1/search?q=keyboard" | jq .

# Filter by country
curl "http://localhost:8080/registry/v1/search?q=keyboard&country=US" | jq .

# Filter by category
curl "http://localhost:8080/registry/v1/search?q=mat&category=sports" | jq .

# Filter by price range (amounts in cents)
curl "http://localhost:8080/registry/v1/search?q=keyboard&price_min=5000&price_max=10000" | jq .

# Filter by currency
curl "http://localhost:8080/registry/v1/search?q=keyboard&currency=USD" | jq .

# Pagination
curl "http://localhost:8080/registry/v1/search?q=keyboard&offset=0&limit=10" | jq .
```

Search results include the store ID and store name, so agents know where to go to purchase:

```json
{
  "data": [
    {
      "id": "prod_001",
      "name": "Wireless Keyboard",
      "description": "Compact mechanical keyboard with Bluetooth",
      "price": { "amount": 7999, "currency": "USD" },
      "category": "electronics",
      "store_id": "str_abc123",
      "store_name": "ACE Demo Store"
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 20
}
```

---

## 14. Payment-as-Auth (Headless Merchant Mode)

ACE supports a headless merchant model where agents can access any endpoint by paying per request, without accounts or API keys. Both authentication modes work simultaneously.

### How It Works

The store declares its payment-as-auth support in `/.well-known/agent-commerce`:

```json
{
  "payment_auth": {
    "enabled": true,
    "header": "X-ACE-Payment",
    "providers": ["x402", "mpp", "mock"],
    "default_currency": "USD"
  }
}
```

### No Auth -- 402 Payment Required

When a request arrives with neither an API key nor a payment token, the store returns HTTP 402 with pricing information:

```bash
curl -i http://localhost:8081/ace/v1/products
```

```
HTTP/1.1 402 Payment Required
X-ACE-Price: 0.00
X-ACE-Currency: USD
Content-Type: application/json

{
  "error": "Payment or API key required",
  "code": "payment_required",
  "pricing": {
    "price": 0,
    "currency": "USD",
    "accepted_providers": ["mock"]
  }
}
```

The 402 response tells the agent exactly what it needs to include in the next request.

### Payment Auth

Include an `X-ACE-Payment` header with the format `provider:token`:

```bash
curl http://localhost:8081/ace/v1/products \
  -H "X-ACE-Payment: mock:any_token_here" | jq .
```

The store validates the payment token against the declared provider. For the `mock` provider, any token is accepted. For production providers (x402, MPP, Stripe), the token is verified against the payment network.

### API Key Auth (still works)

Traditional API key authentication continues to work alongside payment-as-auth:

```bash
curl http://localhost:8081/ace/v1/products \
  -H "X-ACE-Key: <YOUR_DEMO_API_KEY>" | jq .
```

### Check Pricing Before Paying

Query the pricing endpoint to see the cost of each API operation before making a request:

```bash
curl http://localhost:8081/ace/v1/pricing | jq .
```

### Pricing Headers

Every authenticated response includes pricing headers so agents always know the cost:

```
X-ACE-Price: 0.00
X-ACE-Currency: USD
```

This allows agents to track their spending and enforce budget limits programmatically.

---

## 15. Full Architecture Overview

Here is how all the components fit together:

```
                          ACE Protocol -- Full Architecture

    E-COMMERCE PLATFORMS                    ACE LAYER                         CONSUMERS
    ====================                    =========                         =========

    +------------------+
    |  Shopify Store   |---+
    +------------------+   |
                           |    +---------------+
    +------------------+   +--->| ace-connect   |---+
    |  Tiendanube      |------->| (adapters)    |   |
    +------------------+   +--->|               |   |
                           |    +-------+-------+   |
    +------------------+   |            |           |
    |  WooCommerce     |---+   Embedded ACE Server  |
    +------------------+            |               |
                                    |               |
                                    v               |
    +------------------+    +-------+-------+       |       +------------------+
    |  ACE Demo Store  |--->|   Registry    |<------+       |  Claude / GPT    |
    |  (ace-server)    |    |   (:8080)     |               |  (via MCP Buyer) |
    +------------------+    |               |               +--------+---------+
            |               | - Store index |                        |
            |               | - Elasticsearch               +--------+---------+
            |               | - Health checks|              |  mcp-buyer       |
            |               | - Product sync |              |  (13 MCP tools)  |
            |               +-------+-------+               +--------+---------+
            |                       |                                |
            |                       |                                |
            v                       v                                v
    +-------+-------+      +-------+--------+              +--------+---------+
    |  ACE Endpoints |      |  Search API    |              |  ACE HTTP calls  |
    |                |      |                |              |                  |
    | /.well-known/  |      | /registry/v1/  |              | GET /products    |
    |   agent-commerce      |   search?q=... |              | POST /cart       |
    | /ace/v1/products      |   stores       |              | POST /orders     |
    | /ace/v1/cart   |      |   products/sync|              | POST /orders/pay |
    | /ace/v1/orders |      |                |              |                  |
    | /ace/v1/pricing|      +----------------+              +------------------+
    +----------------+
                                                            +------------------+
                                                            |  agent-buyer     |
                                                            |  (Go CLI demo)   |
                                                            |  Autonomous E2E  |
                                                            |  purchase flow   |
                                                            +------------------+
```

**Data flow:**

1. **Store onboarding**: A Shopify store runs `ace-connect`, which fetches products via the Shopify Admin API, starts an ACE-compatible HTTP server, and registers in the registry. Products are synced to Elasticsearch for cross-store search.

2. **Agent discovery**: An AI agent (Claude via MCP, or the Go demo buyer) queries the registry to find stores by category, country, or product keyword. The registry returns store metadata and product search results.

3. **Agent purchase**: The agent connects to a specific store's ACE endpoints, browses the catalog, creates a cart, places an order, and initiates payment. Authentication is via API key or payment-as-auth.

4. **Audit trail**: Every action on the store is logged immutably with correlation IDs. Store owners can review what agents did and configure policies to require approval for sensitive operations.

---

## Troubleshooting

| Problem | Solution |
|---------|---------|
| `connection refused :8080` | Registry is not running. Start it with `cd registry && go run ./cmd/registry` |
| `connection refused :8081` | ACE server is not running. Start it with `cd ace-server && go run ./cmd/ace-server` |
| `401 Unauthorized` | Check you're sending `X-ACE-Key: <YOUR_DEMO_API_KEY>` |
| `402 Payment Required` | Send `X-ACE-Payment: mock:any_token` or use an API key |
| `No stores found in registry` | Register the store first (step 4 above) |
| `go: module not found` | Run from the repo root (where `go.work` is), not from a subdirectory |
| Port already in use | Change port: `PORT=9090 go run ./cmd/ace-server` |
| Elasticsearch not connecting | Check `ELASTICSEARCH_URL` is set and ES is running on port 9200 |
| Search returns no results | Products must be synced first -- see section 13 |
| MCP server not showing in Claude | Check the path in `claude_desktop_config.json` is absolute and the build is up to date |
| ACE Connect fails to register | Make sure the registry is running and reachable at the `--registry` URL |

---

## What's Next

- Read the [full protocol spec](ace-spec/README.md)
- Check the [architecture and business docs](00-negocio.md)
- Open an issue or PR to contribute
- Deploy your own store and register it in the network

The more stores that implement ACE, the more useful buyer agents become -- and the closer we get to a truly open agentic commerce ecosystem.
