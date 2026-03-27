# ACE Protocol -- Architecture

## Overview

ACE (Agent Commerce Exchange) is an open protocol that defines how autonomous buyer agents discover stores, browse catalogs, manage carts, place orders, and initiate payment without a human in the loop. The system consists of six components: a reference ACE server (Go) implementing the store-side protocol, a registry service (Go) backed by Elasticsearch for cross-store discovery and product search, an MCP buyer server (TypeScript) that exposes the protocol as 13 MCP tools for Claude/GPT agents, ACE Connect (TypeScript) that bridges existing e-commerce platforms like Shopify into the ACE ecosystem, a demo CLI buyer agent (Go), and a shared types module consumed by all Go components.

## System Architecture Diagram

```
                                  E-COMMERCE PLATFORMS
                                  --------------------
                                  Shopify API
                                  Tiendanube API (future)
                                  CSV / Manual (future)
                                       |
                                       | adapter.fetchProducts()
                                       v
                              +--------------------+
                              |    ACE Connect     |  TypeScript, node:http
                              |    (ace-connect/)  |
                              |--------------------|
                              | Shopify Adapter    |
                              | Embedded ACE Server|----+
                              | Periodic Sync      |    |
                              +--------------------+    |
                                  |            |        |
              registry token      |            |        |  ACE Buyer API
              + product sync      |            |        |  /.well-known, /ace/v1/*
                                  v            |        |
  +-------------------------------+            |        |
  |           REGISTRY            |            |        |
  |         (registry/)           |            |        |
  |-------------------------------|            |        |
  | Store registration + tokens   |            |        |
  | Product search (Elasticsearch)|            |        |
  | Health monitoring             |            |        |
  | Content moderation            |            |        |
  +-------------------------------+            |        |
       ^              |                        |        |
       |              | search results         |        |
       |              v                        |        |
  +----+---------------------------+           |        |
  |        BUYER AGENTS            |           |        |
  |--------------------------------|           |        |
  |                                |           |        |
  |  MCP Buyer (mcp-buyer/)       |           |        |
  |  TypeScript, 13 MCP tools     |           |        |
  |  Claude / GPT / any MCP host  |-----------+--------+----> Direct HTTP
  |                                |                    |      to stores
  |  Agent Buyer (agent-buyer/)   |                    |
  |  Go CLI demo, 7-step flow    |--------------------+
  |                                |
  +--------------------------------+
                                          |
                                          | GET /.well-known/agent-commerce
                                          | GET/POST /ace/v1/*
                                          v
                              +--------------------+
                              |    ACE Server      |  Go, stdlib net/http
                              |   (ace-server/)    |
                              |--------------------|
                              | Buyer API          |
                              | Admin API          |
                              | Trust Layer        |
                              | Payment Validation |
                              | Audit Log          |
                              +--------------------+
```

```
COMPONENT CONNECTIVITY SUMMARY

  Shopify ----adapter----> ace-connect ----> Embedded ACE Server
                               |
                               +-- registry token --> Registry (Elasticsearch)
                               +-- product sync ----> Registry (Elasticsearch)

  ace-server (reference) ----> standalone store, same ACE API surface

  Registry <---- store registration (ace-server OR ace-connect)
  Registry <---- product sync (push model, bearer token)
  Registry ----> Elasticsearch (stores index + products index)

  MCP Buyer ----> Registry (discover_stores, search_products)
  MCP Buyer ----> Store (browse, cart, order, pay) via direct HTTP

  Agent Buyer --> Registry (discover) --> Store (purchase flow)
```

## Components

### ACE Server (Go)

Reference implementation of the ACE store protocol. Pure Go stdlib (`net/http`), no frameworks.

**Location:** `ace-server/`

**Internal packages:**

| Package | Purpose |
|---------|---------|
| `handlers/` | Buyer API (catalog, cart, orders, payments, shipping, discovery) and Admin API (product CRUD, API key management, policy config) |
| `middleware/` | Dual-mode authentication -- accepts API key (`X-ACE-Key`) or payment token (`X-ACE-Payment`); returns HTTP 402 with pricing info when neither is present |
| `payment/` | `PaymentValidator` interface with mock provider; extensible for Stripe, x402, MPP |
| `policy/` | Trust layer / policy engine -- ALLOW, DENY, APPROVAL effects per action; budget limits per agent |
| `audit/` | Immutable audit logger with correlation IDs; logs every agent action |
| `store/` | In-memory data store for products, carts, orders, API keys |
| `sync/` | Product sync to registry (pushes catalog metadata using registry token) |

**Key features:**
- `GET /.well-known/agent-commerce` discovery endpoint with `payment_auth` declaration
- Payment-as-auth: agents pay per request without accounts
- Per-request pricing model with `X-ACE-Price` / `X-ACE-Currency` response headers
- All sensitive actions gated by configurable policies
- Every operation produces an immutable audit trail entry

### Registry (Go)

Centralized discovery and search service. Stores register themselves and push product metadata. Buyers search across all stores without visiting each one.

**Location:** `registry/`

**Internal packages:**

| Package | Purpose |
|---------|---------|
| `handlers/` | Registry API (store CRUD), search endpoint, product sync endpoint, content moderation (report + delete) |
| `search/` | Elasticsearch integration -- product index with full-text search, filters (category, country, currency, price range, stock), sorting, pagination |
| `auth/` | Registry token generation (`rgt_` prefix, 32-byte hex) and bcrypt validation |
| `healthcheck/` | Periodic health monitoring of registered ACE endpoints |
| `store/` | In-memory registry store + token storage |

**Elasticsearch indexes:**
- `stores` -- registered store metadata (migrated from in-memory)
- `products` -- cross-store product index with composite key `{store_id}::{product_id}`

**Key APIs:**
- `POST /registry/v1/stores` -- register store, receive `registry_token` (returned once, stored as bcrypt hash)
- `POST /registry/v1/products/sync` -- push product metadata (single or batch, max 500 per request)
- `DELETE /registry/v1/products/sync/{product_id}` -- remove product from index
- `GET /registry/v1/search` -- full-text product search with filters and pagination
- `GET /registry/v1/stores` / `GET /registry/v1/stores/{id}` -- store-level queries

Falls back to in-memory storage when Elasticsearch is unavailable (for quick testing without Docker).

### MCP Buyer (TypeScript)

MCP server that exposes ACE buyer operations as 13 tools, making the protocol accessible to any MCP-compatible agent (Claude, GPT, or custom).

**Location:** `mcp-buyer/`

**Source structure:**

| File | Purpose |
|------|---------|
| `index.ts` | MCP server entry point, tool registration |
| `config.ts` | Environment variable parsing and validation |
| `tools/discovery.ts` | `discover_stores`, `search_products` |
| `tools/catalog.ts` | `browse_store`, `get_product`, `get_pricing` |
| `tools/purchase.ts` | `create_cart`, `get_cart`, `add_to_cart`, `shipping_quote`, `place_order`, `pay_order`, `get_order`, `payment_status` |
| `client/registry.ts` | HTTP client for registry API |
| `client/store.ts` | HTTP client for ACE store API with automatic auth header injection |

**Dual mode:**
- **Single-store mode:** `ACE_STORE_URL` points directly to one store
- **Registry mode:** `ACE_REGISTRY_URL` enables cross-store discovery and search

Both modes can be active simultaneously. Auth headers (`X-ACE-Key` or `X-ACE-Payment`) are injected automatically on every request.

**Dependencies:** `@modelcontextprotocol/sdk`, `zod`. HTTP via native `fetch`.

**Distribution:** Published as `ace-buyer-mcp` for `npx` usage.

### ACE Connect (TypeScript)

CLI tool that connects existing e-commerce platforms to ACE with a single command. Uses an adapter pattern -- adding a new platform requires implementing one interface (`EcommerceAdapter`).

**Location:** `ace-connect/`

**Source structure:**

| File | Purpose |
|------|---------|
| `index.ts` | CLI entry point, argument parsing, orchestration, graceful shutdown |
| `adapters/adapter.ts` | `EcommerceAdapter` interface and `AceProduct` / `AceVariant` types |
| `adapters/shopify.ts` | Shopify REST Admin API adapter (product fetch, price conversion, pagination) |
| `server/app.ts` | Embedded ACE-compatible HTTP server (node:http) with full buyer API surface |
| `server/store.ts` | In-memory store for products, carts, orders |
| `server/auth.ts` | Dual auth (mock payment-as-auth by default) |
| `sync.ts` | Periodic sync loop using `adapter.fetchProducts()` |
| `registry.ts` | Registry registration + product push; persists credentials to `.ace-connect.json` |

**Shopify adapter details:**
- Uses REST Admin API `2025-01` (no GraphQL, no SDK)
- Handles pagination via `Link` header for stores with 250+ products
- Rate limit handling: exponential backoff on 429 responses
- Product IDs prefixed with `shp_` for namespacing
- Currency auto-detected from Shopify store settings

**Zero runtime dependencies** beyond Node.js stdlib. No Express, no Fastify, no Shopify SDK.

**Usage:**
```
npx ace-connect shopify --shop mystore.myshopify.com --token shp_xxx --registry http://localhost:8080
```

### Agent Buyer (Go)

Demo CLI that executes the full 7-step purchase flow to validate the protocol end-to-end.

**Location:** `agent-buyer/`

**Steps:**
1. Query registry for stores matching search criteria
2. Discover store via `GET /.well-known/agent-commerce`
3. Browse catalog and select a product
4. Create a cart
5. Add items to cart
6. Place order
7. Initiate payment and print audit trail

### Shared Types (Go)

Common Go types used across `ace-server`, `registry`, and `agent-buyer`.

**Location:** `shared/ace/types.go`

Defines: `Product`, `Variant`, `Money`, `Cart`, `Order`, `Payment`, `WellKnownResponse`, `PaymentAuthConfig`, `PaymentRequiredResponse`, `PricingInfo`, `PricingSchedule`, `EndpointPrice`, `ErrorResponse`, and related types.

## Data Flow Diagrams

### Discovery Flow

```
Agent                          Registry                    Store
  |                               |                          |
  |-- GET /registry/v1/search --->|                          |
  |   ?q=keyboard&country=US      |                          |
  |                               |-- Elasticsearch query -->|
  |<-- product results -----------|                          |
  |   (product_id, store_id,      |                          |
  |    name, price_range, ...)    |                          |
  |                               |                          |
  |-- GET /.well-known/agent-commerce ---------------------->|
  |<-- discovery manifest (ace_base_url, capabilities) ------|
  |                               |                          |
  |-- GET /ace/v1/products/{id} --------------------------->|
  |<-- full product detail (variants, stock, pricing) ------|
```

### Purchase Flow

```
Agent                                          Store
  |                                              |
  |-- POST /ace/v1/cart ----------------------->|
  |<-- { cart_id: "cart_abc" } -----------------|
  |                                              |
  |-- POST /ace/v1/cart/cart_abc/items -------->|
  |   { product_id, quantity, variant_id }       |
  |<-- updated cart with items + total ---------|
  |                                              |
  |-- POST /ace/v1/orders -------------------->|
  |   { cart_id: "cart_abc" }                    |
  |<-- { order_id, status: "pending" } ---------|
  |                                              |
  |-- POST /ace/v1/orders/{id}/pay ----------->|
  |   { provider: "x402" }                      |
  |<-- { payment_url, amount, status } ---------|
  |                                              |
  |-- GET /ace/v1/orders/{id}/pay/status ------>|
  |<-- { status: "completed" } -----------------|
```

### Product Sync Flow

```
Store (or ACE Connect)               Registry                Elasticsearch
  |                                      |                        |
  |-- POST /registry/v1/stores -------->|                        |
  |<-- { store_id, registry_token } ----|                        |
  |                                      |                        |
  |-- POST /registry/v1/products/sync ->|                        |
  |   Authorization: Bearer rgt_xxx      |                        |
  |   { products: [...] }               |                        |
  |                                      |-- validate token       |
  |                                      |   (bcrypt compare)     |
  |                                      |                        |
  |                                      |-- bulk index --------->|
  |                                      |   _id = store::prod    |
  |<-- { indexed: 47, errors: [] } -----|                        |
  |                                      |                        |
  |   ... periodic re-sync ...           |                        |
```

### Payment-as-Auth Flow

```
Agent                                          Store
  |                                              |
  |-- GET /ace/v1/products                       |
  |   (no auth headers)                          |
  |<-- HTTP 402 ---------------------------------|
  |   { pricing: { price: 0.00,                  |
  |     currency: "USD",                         |
  |     accepted_providers: ["mock","x402"] }}   |
  |                                              |
  |-- GET /ace/v1/products                       |
  |   X-ACE-Payment: mock:token123               |
  |<-- HTTP 200 ---------------------------------|
  |   X-ACE-Price: 0.00                          |
  |   X-ACE-Currency: USD                        |
  |   [products...]                              |
```

When both `X-ACE-Key` and `X-ACE-Payment` are present, the API key takes priority and the payment header is ignored. This prevents double-charging.

## Technology Decisions

| Decision | Rationale |
|----------|-----------|
| **Go for backend servers** (ace-server, registry, agent-buyer) | stdlib `net/http` only, no frameworks. Minimal binary, fast startup, straightforward concurrency. |
| **TypeScript for client-facing tools** (mcp-buyer, ace-connect) | MCP SDK is TypeScript-native. `npx` distribution for zero-install usage. |
| **Elasticsearch for product search** | Full-text search with filters, facets, and relevance scoring across all indexed stores. Optional -- registry falls back to in-memory when ES is unavailable. |
| **In-memory data stores** | Simplifies the reference implementation. No database setup required for local development. PostgreSQL persistence planned for Phase 1. |
| **Docker Compose for local dev** | Single `docker-compose up` starts Elasticsearch, registry, and ace-server together. |
| **Zero external runtime dependencies in ace-connect** | Only Node.js stdlib (`node:http`, `node:crypto`, global `fetch`). No Express, no platform SDKs. Keeps the binary small and auditable. |
| **Go workspace (`go.work`)** | Manages multi-module monorepo (ace-server, registry, agent-buyer, shared) without replace directives in individual `go.mod` files. |
| **Adapter pattern in ace-connect** | Adding a new platform (Tiendanube, WooCommerce, CSV) requires implementing one interface. Zero changes to server, sync, or registry code. |

## Security Model

The protocol enforces safety through several layers:

- **Dual authentication:** API key (`X-ACE-Key`) for established relationships, payment-as-auth (`X-ACE-Payment`) for anonymous per-request access
- **Registry tokens:** 32-byte hex with `rgt_` prefix, bcrypt-hashed at rest, issued once at store registration
- **Policy engine:** configurable ALLOW / DENY / APPROVAL effects per action; sensitive operations require explicit opt-in
- **Budget limits:** per-agent spending caps enforced by the trust layer
- **Immutable audit log:** every agent action recorded with correlation ID, actor, action, resource, and timestamp
- **Admin authentication:** separate admin token for store management operations

For the complete security model, threat analysis, and governance policies, see [SECURITY.md](SECURITY.md).
