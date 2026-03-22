# Scalable Search for ACE Protocol

**Date:** 2026-03-21
**Status:** Approved

## Problem

The current ACE protocol has no centralized product search. The registry stores are in-memory with naive `strings.Contains` matching. Buyers must know which store to visit — there's no way to search across stores. This doesn't scale beyond a demo.

## Decision

Implement a hybrid search architecture where:
- Stores push lightweight product metadata to a centralized index
- Buyers search the centralized index to discover products across all stores
- Full product details (variants, exact stock) remain in each store

## Architecture

```
BUYER AGENT
  │
  ├─① GET /registry/v1/search?q=keyboard&country=US
  │     → Elasticsearch query across all indexed products
  │     → Returns: product summaries with store_id
  │
  └─② Connect to chosen store
        GET /.well-known/agent-commerce → ace_base_url
        GET /ace/v1/products/{id}      → full detail + variants
        POST /ace/v1/cart              → normal purchase flow

STORE (ACE SERVER)
  │
  ├─ POST /registry/v1/stores           → register, receive registry_token
  ├─ POST /registry/v1/products/sync    → push product metadata (uses registry_token)
  └─ DELETE /registry/v1/products/sync/{product_id} → remove from index

REGISTRY
  │
  └─ Elasticsearch
       ├─ index: stores    (migrated from in-memory)
       └─ index: products  (new centralized product index)
```

## Data Model

### Document ID Strategy

Elasticsearch documents use a composite key: `{store_id}::{product_id}` as the document `_id`. This ensures product IDs are unique across stores — two stores can both have a `prod_001` without collision. All delete/update operations use this composite key, with `store_id` inferred from the `registry_token`.

### Price Range Derivation

The store computes `price_range` before syncing:
- If a product has no variants: `min == max == product.Price.Amount`
- If a product has variants: `min` is the lowest variant price, `max` is the highest
- `currency` is the product's primary currency. Multi-currency products should sync one entry per currency (future extension; for now, use the store's primary currency)

### In-Stock Derivation

A product is `in_stock: true` if at least one variant has `inventory > 0`. If the product has no variants, it is always `in_stock: true` (stock tracking is variant-level). Stores must re-sync the product when stock status changes (any variant goes from >0 to 0 or vice versa).

### Search-Only Fields

The fields `category`, `tags`, `rating`, and `location` are search-index metadata provided during sync. They do NOT need to exist on the `ace.Product` shared type. Stores provide them based on their own internal data. This keeps the shared type unchanged and avoids coupling the protocol to search concerns.

### Product Index Schema (Elasticsearch)

```json
{
  "product_id": "string (keyword)",
  "store_id": "string (keyword)",
  "store_name": "string (text)",
  "name": "string (text, analyzed)",
  "description": "string (text, analyzed)",
  "category": "string (keyword)",
  "tags": ["string (keyword)"],
  "price_range": {
    "min": "long",
    "max": "long",
    "currency": "keyword"
  },
  "variants_summary": ["string (keyword)"],
  "image_url": "string (keyword)",
  "in_stock": "boolean",
  "rating": {
    "average": "float",
    "count": "integer"
  },
  "location": {
    "country": "keyword",
    "region": "keyword"
  },
  "updated_at": "date"
}
```

### Store Index Schema (Elasticsearch)

Same fields as current `StoreEntry`, migrated from in-memory map to Elasticsearch.

### Registry Token Storage

The `registry_token` is a random 32-byte hex string with `rgt_` prefix, generated at store registration. It is stored as a **bcrypt hash** alongside the `StoreEntry` (in a separate `registry_tokens` map/table, NOT in the Elasticsearch index). The plaintext token is returned ONCE in the registration response and never stored.

- **No rotation for MVP.** If a store loses its token, it must re-register (new store ID).
- **Revocation:** Deleting a store from the registry invalidates its token and removes all its indexed products.
- **Implementation:** A new `StoreRegistrationResponse` type wraps `StoreEntry` + `registry_token` for the registration response only.

## API Changes

### Modified: `POST /registry/v1/stores`

Now returns a `registry_token` in the response:

```json
{
  "id": "str_abc123",
  "name": "My Store",
  "registry_token": "rgt_...",
  "...existing fields..."
}
```

The token is generated once at registration and used for all product sync operations.

### New: `POST /registry/v1/products/sync`

Push product metadata to the centralized index. Accepts a single product or a batch.

**Headers:**
- `Authorization: Bearer <registry_token>`

**Body (single):**
```json
{
  "product_id": "prod_abc123",
  "name": "Mechanical Keyboard RGB",
  "description": "Cherry MX switches, hot-swappable...",
  "category": "electronics",
  "tags": ["keyboard", "mechanical", "gaming"],
  "price_range": { "min": 6999, "max": 8999, "currency": "USD" },
  "variants_summary": ["Red", "Blue", "Black"],
  "image_url": "https://store.example.com/img/kb.jpg",
  "in_stock": true,
  "rating": { "average": 4.3, "count": 128 },
  "location": { "country": "US", "region": "NA" }
}
```

**Body (batch):**
```json
{
  "products": [
    { "product_id": "...", "...": "..." },
    { "product_id": "...", "...": "..." }
  ]
}
```

The `store_id` and `store_name` are inferred from the `registry_token` — stores cannot index products under another store's ID.

**Response (200 OK):**
```json
{
  "indexed": 3,
  "updated": 2,
  "errors": [
    { "product_id": "prod_bad", "error": "missing required field: name" }
  ]
}
```

Partial failures return `200` with error details per product. The successfully synced products are indexed regardless of individual failures. All errors follow the existing `ace.ErrorResponse` pattern (`error`, `code`, `details` fields).

### New: `DELETE /registry/v1/products/sync/{product_id}`

Remove a product from the centralized index. The `store_id` is inferred from the `registry_token` — a store can only delete its own products. Internally deletes the Elasticsearch document with `_id = {store_id}::{product_id}`.

**Headers:**
- `Authorization: Bearer <registry_token>`

**Response:** `204 No Content`

### New: `GET /registry/v1/search`

Unified product search across all stores.

**Query Parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `q` | string | — | Full-text search on name + description + tags |
| `category` | string | — | Exact match filter |
| `country` | string | — | Exact match filter |
| `currency` | string | — | Exact match filter |
| `price_min` | int | — | Range filter (min price in cents) |
| `price_max` | int | — | Range filter (max price in cents) |
| `in_stock` | bool | `true` | Only show in-stock products |
| `sort` | string | `relevance` | `relevance`, `price_asc` (by min), `price_desc` (by max), `rating` (by average desc) |
| `offset` | int | `0` | Pagination offset |
| `limit` | int | `20` | Pagination limit (max 100) |

**Response:**
```json
{
  "data": [
    {
      "product_id": "prod_abc123",
      "store_id": "str_xyz789",
      "store_name": "Acme Electronics",
      "name": "Mechanical Keyboard RGB",
      "description": "Cherry MX switches...",
      "category": "electronics",
      "tags": ["keyboard", "mechanical"],
      "price_range": { "min": 6999, "max": 8999, "currency": "USD" },
      "variants_summary": ["Red", "Blue", "Black"],
      "image_url": "https://...",
      "in_stock": true,
      "rating": { "average": 4.3, "count": 128 },
      "location": { "country": "US", "region": "NA" }
    }
  ],
  "total": 1523,
  "offset": 0,
  "limit": 20
}
```

## What Doesn't Change

- The purchase flow (cart → order → pay) remains direct between buyer and store
- Each store owns its complete catalog with full variant/stock details
- The `.well-known/agent-commerce` endpoint is unchanged
- The Admin API on each store is unchanged
- Existing `GET /registry/v1/stores` and `GET /registry/v1/stores/{id}` remain for store-level queries

## Authentication

- **Store → Registry (sync):** `registry_token` issued at store registration. Bearer token in Authorization header.
- **Buyer → Registry (search):** No auth required for search (public index). Rate limiting applied per IP.
- **Buyer → Store (purchase):** Existing `X-ACE-Key` mechanism, unchanged.

## Scaling Considerations

- Elasticsearch sharding by `location.region` for geo-distributed queries
- Bulk sync endpoint allows stores to push entire catalog in batches (max 500 products per request)
- `updated_at` is server-set (registry sets it on ingestion) to prevent clock skew issues
- `in_stock` filter avoids showing unavailable products without real-time stock checks
- Sync rate limit: 100 requests/minute per `registry_token`
- Store deregistration cascades: deleting a store removes all its indexed products
- `store_name` is denormalized in the product index; store name changes do NOT propagate to existing products (accepted trade-off, reconciled on next product sync)
- Future: add a pull-based reconciliation job to catch stale/orphaned products

## Local Development

For local development, Elasticsearch runs via Docker. A `docker-compose.yml` at the project root provides Elasticsearch + the registry + ace-server. The registry falls back to in-memory storage if Elasticsearch is unavailable (for quick testing without Docker).

## Future Extensions (Not in Scope)

- Semantic/vector search layer on top of Elasticsearch
- Federated search as alternative to centralized index
- Pull-based crawling for reconciliation
- Product recommendations engine
- Search analytics and ranking optimization
