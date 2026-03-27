# API Reference

Complete reference for the ACE (Agent Commerce Engine) protocol endpoints, covering ACE Store buyer and admin APIs, and the Registry API.

All request and response bodies use `Content-Type: application/json`.

---

## Table of Contents

- [ACE Store API -- Buyer Endpoints](#ace-store-api----buyer-endpoints)
- [ACE Store API -- Admin Endpoints](#ace-store-api----admin-endpoints)
- [Registry API](#registry-api)
- [Authentication](#authentication)
- [Response Headers](#response-headers)
- [Pagination](#pagination)
- [Error Format](#error-format)

---

## ACE Store API -- Buyer Endpoints

Base URL: `https://<store-host>`

### GET /.well-known/agent-commerce

Discovery endpoint. Returns store capabilities, auth configuration, and supported currencies.

| Field | Value |
|-------|-------|
| Auth  | None  |

**Response** `200 OK`:

```json
{
  "store_id": "string",
  "name": "string",
  "version": "1.0.0",
  "ace_base_url": "https://<store-host>/ace/v1",
  "capabilities": ["catalog", "cart", "orders", "payments", "shipping"],
  "auth": {
    "type": "api_key",
    "header": "X-ACE-Key"
  },
  "currencies": ["USD"],
  "payment_auth": {
    "enabled": true,
    "header": "X-ACE-Payment",
    "providers": ["stripe", "mercadopago"],
    "default_currency": "USD"
  },
  "policies_public": {}
}
```

```bash
curl https://localhost:8080/.well-known/agent-commerce
```

---

### GET /ace/v1/pricing

Returns the pricing schedule for all buyer endpoints.

| Field | Value |
|-------|-------|
| Auth  | None  |

**Response** `200 OK`:

```json
{
  "default_currency": "USD",
  "endpoints": [
    { "method": "GET",  "path": "/ace/v1/products",          "price": 0.00 },
    { "method": "GET",  "path": "/ace/v1/products/{id}",     "price": 0.00 },
    { "method": "POST", "path": "/ace/v1/cart",              "price": 0.00 },
    { "method": "POST", "path": "/ace/v1/cart/{id}/items",   "price": 0.00 },
    { "method": "POST", "path": "/ace/v1/orders",            "price": 0.00 },
    { "method": "POST", "path": "/ace/v1/orders/{id}/pay",   "price": 0.00 }
  ]
}
```

```bash
curl https://localhost:8080/ace/v1/pricing
```

---

### GET /ace/v1/products

List published products with optional search and filtering.

| Field | Value |
|-------|-------|
| Auth  | API key or payment-as-auth |

**Query Parameters**:

| Parameter  | Type   | Default | Description |
|------------|--------|---------|-------------|
| `q`        | string | --      | Free-text search query |
| `category` | string | --      | Filter by category |
| `offset`   | int    | 0       | Pagination offset |
| `limit`    | int    | 20      | Page size (max 100) |

**Response** `200 OK`:

```json
{
  "data": [
    {
      "id": "string",
      "name": "string",
      "description": "string",
      "price": { "amount": 1999, "currency": "USD" },
      "variants": [],
      "status": "published",
      "pricing_model": "",
      "price_per_request": 0,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 42,
  "offset": 0,
  "limit": 20
}
```

```bash
curl -H "X-ACE-Key: <key>" "https://localhost:8080/ace/v1/products?q=widget&limit=10"
```

---

### GET /ace/v1/products/{id}

Get a single published product by ID.

| Field | Value |
|-------|-------|
| Auth  | API key or payment-as-auth |

**Path Parameters**:

| Parameter | Type   | Description |
|-----------|--------|-------------|
| `id`      | string | Product ID  |

**Response** `200 OK`: A single `Product` object (same schema as the list item above).

**Response** `404 Not Found`: Product does not exist or is not published.

```bash
curl -H "X-ACE-Key: <key>" https://localhost:8080/ace/v1/products/prod_abc123
```

---

### POST /ace/v1/cart

Create a new empty cart.

| Field | Value |
|-------|-------|
| Auth  | API key or payment-as-auth |

**Request body**: None.

**Response** `201 Created`:

```json
{
  "id": "string",
  "items": [],
  "total": { "amount": 0, "currency": "USD" },
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

```bash
curl -X POST -H "X-ACE-Key: <key>" https://localhost:8080/ace/v1/cart
```

---

### POST /ace/v1/cart/{id}/items

Add an item to an existing cart.

| Field | Value |
|-------|-------|
| Auth  | API key or payment-as-auth |

**Path Parameters**:

| Parameter | Type   | Description |
|-----------|--------|-------------|
| `id`      | string | Cart ID     |

**Request body**:

```json
{
  "product_id": "string",
  "variant_id": "string (optional)",
  "quantity": 1
}
```

**Response** `200 OK`: The updated `Cart` object.

**Error responses**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `invalid_body` | Malformed JSON |
| 400 | `invalid_quantity` | Quantity <= 0 |
| 400 | `invalid_pricing_model` | Per-request product cannot be carted |
| 404 | `product_not_found` | Product missing or unpublished |
| 404 | `variant_not_found` | Variant ID does not exist |
| 404 | `cart_not_found` | Cart ID does not exist |
| 409 | `insufficient_stock` | Not enough inventory |

```bash
curl -X POST -H "X-ACE-Key: <key>" \
  -H "Content-Type: application/json" \
  -d '{"product_id":"prod_abc123","quantity":2}' \
  https://localhost:8080/ace/v1/cart/cart_xyz/items
```

---

### GET /ace/v1/cart/{id}

Retrieve a cart by ID.

| Field | Value |
|-------|-------|
| Auth  | API key or payment-as-auth |

**Path Parameters**:

| Parameter | Type   | Description |
|-----------|--------|-------------|
| `id`      | string | Cart ID     |

**Response** `200 OK`: A `Cart` object.

**Response** `404 Not Found`: Cart does not exist.

```bash
curl -H "X-ACE-Key: <key>" https://localhost:8080/ace/v1/cart/cart_xyz
```

---

### POST /ace/v1/orders

Create an order from a cart. Decrements inventory for all items.

| Field | Value |
|-------|-------|
| Auth  | API key or payment-as-auth |

**Request body**:

```json
{
  "cart_id": "string"
}
```

**Response** `201 Created`:

```json
{
  "id": "string",
  "cart_id": "string",
  "items": [
    {
      "product_id": "string",
      "product_name": "string",
      "variant_id": "string",
      "quantity": 1,
      "price": { "amount": 1999, "currency": "USD" }
    }
  ],
  "total": { "amount": 1999, "currency": "USD" },
  "status": "pending",
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

**Error responses**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `empty_cart` | Cart has no items |
| 404 | `cart_not_found` | Cart does not exist |
| 409 | `product_unavailable` | Product no longer exists |
| 409 | `insufficient_stock` | Not enough inventory |

```bash
curl -X POST -H "X-ACE-Key: <key>" \
  -H "Content-Type: application/json" \
  -d '{"cart_id":"cart_xyz"}' \
  https://localhost:8080/ace/v1/orders
```

---

### GET /ace/v1/orders/{id}

Retrieve an order by ID.

| Field | Value |
|-------|-------|
| Auth  | API key or payment-as-auth |

**Path Parameters**:

| Parameter | Type   | Description |
|-----------|--------|-------------|
| `id`      | string | Order ID    |

**Response** `200 OK`: An `Order` object.

**Response** `404 Not Found`: Order does not exist.

```bash
curl -H "X-ACE-Key: <key>" https://localhost:8080/ace/v1/orders/ord_abc123
```

---

### POST /ace/v1/orders/{id}/pay

Initiate payment for a pending order. In the current implementation, payments are auto-completed (mock provider).

| Field | Value |
|-------|-------|
| Auth  | API key or payment-as-auth |

**Path Parameters**:

| Parameter | Type   | Description |
|-----------|--------|-------------|
| `id`      | string | Order ID    |

**Request body**:

```json
{
  "provider": "string (optional, defaults to \"mock\")"
}
```

**Response** `201 Created`:

```json
{
  "id": "string",
  "order_id": "string",
  "status": "completed",
  "provider": "mock",
  "amount": { "amount": 1999, "currency": "USD" },
  "external_id": "mock_ext_ord_abc123",
  "payment_url": "https://pay.example.com/mock/ord_abc123",
  "created_at": "2025-01-01T00:00:00Z"
}
```

**Error responses**:

| Status | Code | Condition |
|--------|------|-----------|
| 404 | `not_found` | Order does not exist |
| 409 | `invalid_status` | Order is not in `pending` status |

```bash
curl -X POST -H "X-ACE-Key: <key>" \
  -H "Content-Type: application/json" \
  -d '{"provider":"mock"}' \
  https://localhost:8080/ace/v1/orders/ord_abc123/pay
```

---

### GET /ace/v1/orders/{id}/pay/status

Check payment status for an order.

| Field | Value |
|-------|-------|
| Auth  | API key or payment-as-auth |

**Path Parameters**:

| Parameter | Type   | Description |
|-----------|--------|-------------|
| `id`      | string | Order ID    |

**Response** `200 OK`: A `Payment` object.

**Response** `404 Not Found`: No payment exists for this order.

```bash
curl -H "X-ACE-Key: <key>" https://localhost:8080/ace/v1/orders/ord_abc123/pay/status
```

---

### POST /ace/v1/shipping/quote

Get available shipping options for a set of items and destination.

| Field | Value |
|-------|-------|
| Auth  | API key or payment-as-auth |

**Request body**:

```json
{
  "items": [
    {
      "product_id": "string",
      "variant_id": "string",
      "quantity": 1,
      "price": { "amount": 1999, "currency": "USD" }
    }
  ],
  "destination": {
    "country": "US",
    "state": "CA",
    "city": "San Francisco",
    "postal_code": "94105",
    "line1": "123 Main St",
    "line2": "Apt 4"
  }
}
```

**Response** `200 OK`:

```json
{
  "options": [
    { "id": "ship_standard",  "name": "Standard Shipping",  "price": { "amount": 599,  "currency": "USD" }, "estimated_days": 7 },
    { "id": "ship_express",   "name": "Express Shipping",   "price": { "amount": 1299, "currency": "USD" }, "estimated_days": 3 },
    { "id": "ship_overnight", "name": "Overnight Shipping",  "price": { "amount": 2499, "currency": "USD" }, "estimated_days": 1 }
  ]
}
```

```bash
curl -X POST -H "X-ACE-Key: <key>" \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":"prod_abc123","quantity":1,"price":{"amount":1999,"currency":"USD"}}],"destination":{"country":"US","state":"CA","city":"San Francisco","postal_code":"94105","line1":"123 Main St"}}' \
  https://localhost:8080/ace/v1/shipping/quote
```

---

## ACE Store API -- Admin Endpoints

Base URL: `https://<store-host>`

All admin endpoints require `Authorization: Bearer <ADMIN_TOKEN>` and are scoped to a store via the `{store_id}` path parameter.

---

### POST /api/v1/stores/{store_id}/products

Create a new product. Status is always set to `draft` on creation.

**Request body**:

```json
{
  "name": "string",
  "description": "string",
  "price": { "amount": 1999, "currency": "USD" },
  "variants": [
    {
      "id": "string",
      "name": "string",
      "sku": "string",
      "price": { "amount": 1999, "currency": "USD" },
      "inventory": 100,
      "attributes": { "color": "red" }
    }
  ],
  "pricing_model": "string (optional)",
  "price_per_request": 0.0
}
```

**Response** `201 Created`: The created `Product` object with `status: "draft"`.

```bash
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Widget","description":"A fine widget","price":{"amount":1999,"currency":"USD"}}' \
  https://localhost:8080/api/v1/stores/store_001/products
```

---

### GET /api/v1/stores/{store_id}/products

List all products (any status) with optional filtering.

**Query Parameters**:

| Parameter  | Type   | Default | Description |
|------------|--------|---------|-------------|
| `status`   | string | --      | Filter by status: `draft`, `published`, `unpublished` |
| `category` | string | --      | Filter by category |
| `q`        | string | --      | Free-text search |
| `offset`   | int    | 0       | Pagination offset |
| `limit`    | int    | 20      | Page size (max 100) |

**Response** `200 OK`: `PaginatedResponse<Product>`.

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "https://localhost:8080/api/v1/stores/store_001/products?status=draft&limit=50"
```

---

### PATCH /api/v1/stores/{store_id}/products/{id}

Partially update a product. Send only the fields to change.

**Request body** (all fields optional):

```json
{
  "name": "New Name",
  "description": "Updated description",
  "price": { "amount": 2499, "currency": "USD" },
  "variants": [ ... ]
}
```

**Response** `200 OK`: The updated `Product` object.

**Response** `404 Not Found`: Product does not exist.

```bash
curl -X PATCH -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Widget Pro"}' \
  https://localhost:8080/api/v1/stores/store_001/products/prod_abc123
```

---

### DELETE /api/v1/stores/{store_id}/products/{id}

Delete a product.

**Response** `204 No Content`: Product deleted.

**Response** `404 Not Found`: Product does not exist.

```bash
curl -X DELETE -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/products/prod_abc123
```

---

### POST /api/v1/stores/{store_id}/products/{id}/publish

Publish a product, making it visible to buyers. Subject to policy checks -- may return an approval request if the store has an approval policy for `product.publish`.

**Request body**: None.

**Responses**:

| Status | Condition |
|--------|-----------|
| 200 OK | Product published |
| 202 Accepted | Approval required; returns an `Approval` object |
| 403 Forbidden | Denied by policy (`policy_denied`) |
| 404 Not Found | Product does not exist |

```bash
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/products/prod_abc123/publish
```

---

### POST /api/v1/stores/{store_id}/products/{id}/unpublish

Unpublish a product, hiding it from buyers.

**Request body**: None.

**Response** `200 OK`: The updated `Product` with `status: "unpublished"`.

**Response** `404 Not Found`: Product does not exist.

```bash
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/products/prod_abc123/unpublish
```

---

### GET /api/v1/stores/{store_id}/orders

List all orders for the store.

**Query Parameters**:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `offset`  | int  | 0       | Pagination offset |
| `limit`   | int  | 20      | Page size |

**Response** `200 OK`: `PaginatedResponse<Order>`.

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "https://localhost:8080/api/v1/stores/store_001/orders?limit=10"
```

---

### POST /api/v1/stores/{store_id}/orders/{id}/fulfill

Mark an order as fulfilled. The order must be in `paid` status.

**Request body**: None.

**Response** `200 OK`: The updated `Order` with `status: "fulfilled"`.

**Error responses**:

| Status | Code | Condition |
|--------|------|-----------|
| 404 | `not_found` | Order does not exist |
| 409 | `invalid_status` | Order is not `paid` |

```bash
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/orders/ord_abc123/fulfill
```

---

### POST /api/v1/stores/{store_id}/orders/{id}/refund

Refund an order. Subject to policy checks.

**Request body**: None.

**Responses**:

| Status | Condition |
|--------|-----------|
| 200 OK | Order refunded |
| 202 Accepted | Approval required; returns an `Approval` object |
| 403 Forbidden | Denied by policy |
| 404 Not Found | Order does not exist |

```bash
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/orders/ord_abc123/refund
```

---

### GET /api/v1/stores/{store_id}/policies

Retrieve the current policy set for the store.

**Response** `200 OK`:

```json
[
  {
    "id": "string",
    "action": "product.publish",
    "effect": "allow"
  },
  {
    "id": "string",
    "action": "order.refund",
    "effect": "approval"
  }
]
```

Possible `effect` values: `allow`, `deny`, `approval`.

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/policies
```

---

### PUT /api/v1/stores/{store_id}/policies

Replace the entire policy set for the store.

**Request body**: An array of `Policy` objects.

```json
[
  { "id": "pol_1", "action": "product.publish", "effect": "approval" },
  { "id": "pol_2", "action": "order.refund", "effect": "deny" }
]
```

**Response** `200 OK`: The updated policy array.

```bash
curl -X PUT -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '[{"id":"pol_1","action":"product.publish","effect":"approval"}]' \
  https://localhost:8080/api/v1/stores/store_001/policies
```

---

### GET /api/v1/stores/{store_id}/approvals

List all pending approval requests.

**Response** `200 OK`: An array of `Approval` objects.

```json
[
  {
    "id": "string",
    "action": "product.publish",
    "resource": "prod_abc123",
    "status": "pending",
    "requested_by": "agent:bot-1",
    "resolved_by": "",
    "created_at": "2025-01-01T00:00:00Z",
    "resolved_at": null
  }
]
```

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/approvals
```

---

### POST /api/v1/stores/{store_id}/approvals/{id}/approve

Approve a pending approval request.

**Request body**: None.

**Response** `200 OK`: The resolved `Approval` object with `status: "approved"`.

**Response** `404 Not Found`: Approval not found.

```bash
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/approvals/appr_xyz/approve
```

---

### POST /api/v1/stores/{store_id}/approvals/{id}/reject

Reject a pending approval request.

**Request body**: None.

**Response** `200 OK`: The resolved `Approval` object with `status: "rejected"`.

**Response** `404 Not Found`: Approval not found.

```bash
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/approvals/appr_xyz/reject
```

---

### GET /api/v1/stores/{store_id}/audit-logs

Query the audit log for the store.

**Query Parameters**:

| Parameter | Type   | Default | Description |
|-----------|--------|---------|-------------|
| `action`  | string | --      | Filter by action (e.g. `product.publish`) |
| `actor`   | string | --      | Filter by actor identifier |
| `offset`  | int    | 0       | Pagination offset |
| `limit`   | int    | 50      | Page size |

**Response** `200 OK`: `PaginatedResponse<AuditEntry>`.

```json
{
  "data": [
    {
      "id": "string",
      "store_id": "string",
      "action": "product.create",
      "actor": "admin:user@example.com",
      "actor_type": "human",
      "resource": "product",
      "resource_id": "prod_abc123",
      "details": {},
      "correlation_id": "string",
      "timestamp": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 100,
  "offset": 0,
  "limit": 50
}
```

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  "https://localhost:8080/api/v1/stores/store_001/audit-logs?action=product.create&limit=10"
```

---

### POST /api/v1/stores/{store_id}/api-keys

Create a new API key for buyer access.

**Request body**:

```json
{
  "name": "string (required)",
  "scopes": ["catalog", "cart", "orders"]
}
```

**Response** `201 Created`:

```json
{
  "id": "string",
  "name": "string",
  "key_prefix": "ace_k_ab",
  "scopes": ["catalog", "cart", "orders"],
  "created_at": "2025-01-01T00:00:00Z",
  "expires_at": null,
  "key": "ace_k_abcdef1234567890"
}
```

The `key` field is returned only at creation time. Store it securely.

```bash
curl -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Agent Bot Key","scopes":["catalog","cart","orders"]}' \
  https://localhost:8080/api/v1/stores/store_001/api-keys
```

---

### GET /api/v1/stores/{store_id}/api-keys

List all API keys (without the secret key value).

**Response** `200 OK`: An array of `APIKey` objects.

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/api-keys
```

---

### DELETE /api/v1/stores/{store_id}/api-keys/{id}

Revoke an API key.

**Response** `204 No Content`: Key deleted.

**Response** `404 Not Found`: Key does not exist.

```bash
curl -X DELETE -H "Authorization: Bearer $ADMIN_TOKEN" \
  https://localhost:8080/api/v1/stores/store_001/api-keys/key_abc123
```

---

## Registry API

Base URL: `https://<registry-host>`

The Registry is a central directory where stores register themselves and products are indexed for cross-store search.

---

### POST /registry/v1/stores

Register a new store in the registry. The registry fetches the store's `/.well-known/agent-commerce` endpoint to validate it and extract metadata.

| Field | Value |
|-------|-------|
| Auth  | None  |

**Request body**:

```json
{
  "well_known_url": "https://my-store.example.com/.well-known/agent-commerce",
  "categories": ["electronics", "gadgets"],
  "country": "US"
}
```

**Response** `201 Created`:

```json
{
  "id": "string",
  "well_known_url": "https://my-store.example.com/.well-known/agent-commerce",
  "name": "My Store",
  "categories": ["electronics", "gadgets"],
  "country": "US",
  "currencies": ["USD"],
  "capabilities": ["catalog", "cart", "orders", "payments", "shipping"],
  "health_status": "healthy",
  "last_checked": "2025-01-01T00:00:00Z",
  "registered_at": "2025-01-01T00:00:00Z",
  "registry_token": "regt_<random>"
}
```

The `registry_token` is returned only once. It is required for product sync operations.

**Error responses**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `missing_field` | `well_known_url` not provided |
| 400 | `invalid_store` | Could not fetch or validate the well-known URL |

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"well_known_url":"https://my-store.example.com/.well-known/agent-commerce","categories":["electronics"],"country":"US"}' \
  https://localhost:9090/registry/v1/stores
```

---

### GET /registry/v1/stores

List registered stores with optional filters.

| Field | Value |
|-------|-------|
| Auth  | None  |

**Query Parameters**:

| Parameter  | Type   | Default | Description |
|------------|--------|---------|-------------|
| `q`        | string | --      | Free-text search on store name |
| `category` | string | --      | Filter by category |
| `country`  | string | --      | Filter by country code |
| `currency` | string | --      | Filter by supported currency |
| `offset`   | int    | 0       | Pagination offset |
| `limit`    | int    | 20      | Page size |

**Response** `200 OK`: `PaginatedResponse<StoreEntry>`.

```bash
curl "https://localhost:9090/registry/v1/stores?category=electronics&country=US"
```

---

### GET /registry/v1/stores/{id}

Get a single store by ID.

| Field | Value |
|-------|-------|
| Auth  | None  |

**Response** `200 OK`: A `StoreEntry` object.

**Response** `404 Not Found`: Store not found.

```bash
curl https://localhost:9090/registry/v1/stores/store_abc123
```

---

### GET /registry/v1/stores/{id}/health

Trigger a live health check by fetching the store's well-known URL.

| Field | Value |
|-------|-------|
| Auth  | None  |

**Response** `200 OK`: The updated `StoreEntry` with refreshed `health_status` and `last_checked`.

Possible `health_status` values: `healthy`, `degraded`, `down`, `unknown`.

**Response** `404 Not Found`: Store not found.

```bash
curl https://localhost:9090/registry/v1/stores/store_abc123/health
```

---

### POST /registry/v1/stores/{id}/report

Report a store for abuse or policy violations.

| Field | Value |
|-------|-------|
| Auth  | None  |

**Request body**:

```json
{
  "reason": "string (required)",
  "details": "string (optional)"
}
```

**Response** `200 OK`:

```json
{ "status": "reported" }
```

**Error responses**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `missing_field` | `reason` not provided |
| 404 | `not_found` | Store not found |

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"reason":"spam","details":"Listing fake products"}' \
  https://localhost:9090/registry/v1/stores/store_abc123/report
```

---

### DELETE /registry/v1/stores/{id}

Remove a store from the registry. Also deletes all indexed products for that store.

| Field | Value |
|-------|-------|
| Auth  | `Authorization: Bearer <REGISTRY_ADMIN_TOKEN>` |

**Response** `204 No Content`: Store deleted.

**Error responses**:

| Status | Code | Condition |
|--------|------|-----------|
| 401 | `unauthorized` | Missing or invalid admin token |
| 404 | `not_found` | Store not found |

```bash
curl -X DELETE -H "Authorization: Bearer $REGISTRY_ADMIN_TOKEN" \
  https://localhost:9090/registry/v1/stores/store_abc123
```

---

### GET /registry/v1/search

Search products across all registered stores using Elasticsearch.

| Field | Value |
|-------|-------|
| Auth  | None  |

**Query Parameters**:

| Parameter   | Type   | Default | Description |
|-------------|--------|---------|-------------|
| `q`         | string | --      | Free-text search query |
| `category`  | string | --      | Filter by category |
| `country`   | string | --      | Filter by store country |
| `currency`  | string | --      | Filter by price currency |
| `price_min` | int64  | 0       | Minimum price in cents |
| `price_max` | int64  | 0       | Maximum price in cents (0 = no limit) |
| `in_stock`  | bool   | true    | Filter by stock availability |
| `sort`      | string | --      | Sort field |
| `offset`    | int    | 0       | Pagination offset |
| `limit`     | int    | 20      | Page size (max 100) |

**Response** `200 OK`: `PaginatedResponse<ProductSearchResult>`.

```json
{
  "data": [
    {
      "product_id": "string",
      "store_id": "string",
      "store_name": "string",
      "name": "string",
      "description": "string",
      "category": "string",
      "tags": ["string"],
      "price_range": { "min": 1000, "max": 2500, "currency": "USD" },
      "variants_summary": ["Red", "Blue"],
      "image_url": "string",
      "in_stock": true,
      "rating": { "average": 4.5, "count": 128 },
      "location": { "country": "US", "region": "CA" }
    }
  ],
  "total": 350,
  "offset": 0,
  "limit": 20
}
```

```bash
curl "https://localhost:9090/registry/v1/search?q=widget&category=electronics&price_min=1000&limit=10"
```

---

### POST /registry/v1/products/sync

Push products to the registry search index. Accepts a single product or a batch.

| Field | Value |
|-------|-------|
| Auth  | `Authorization: Bearer <REGISTRY_TOKEN>` (from store registration) |

**Single product request**:

```json
{
  "product_id": "string (required)",
  "name": "string",
  "description": "string",
  "category": "string",
  "tags": ["string"],
  "price_range": { "min": 1000, "max": 2500, "currency": "USD" },
  "variants_summary": ["Red", "Blue"],
  "image_url": "string",
  "in_stock": true,
  "rating": { "average": 4.5, "count": 128 },
  "location": { "country": "US", "region": "CA" }
}
```

**Batch request**:

```json
{
  "products": [
    { "product_id": "prod_1", "name": "...", ... },
    { "product_id": "prod_2", "name": "...", ... }
  ]
}
```

**Response** `200 OK`:

```json
{
  "indexed": 2,
  "updated": 0,
  "errors": [
    { "product_id": "", "error": "product_id is required" }
  ]
}
```

**Response** `401 Unauthorized`: Invalid or missing registry token.

```bash
curl -X POST -H "Authorization: Bearer $REGISTRY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id":"prod_1","name":"Widget","description":"A widget","category":"electronics","price_range":{"min":1999,"max":1999,"currency":"USD"},"in_stock":true,"location":{"country":"US","region":"CA"}}' \
  https://localhost:9090/registry/v1/products/sync
```

---

### DELETE /registry/v1/products/sync/{product_id}

Remove a single product from the search index.

| Field | Value |
|-------|-------|
| Auth  | `Authorization: Bearer <REGISTRY_TOKEN>` |

**Path Parameters**:

| Parameter    | Type   | Description |
|--------------|--------|-------------|
| `product_id` | string | Product ID to remove |

**Response** `204 No Content`: Product removed from index.

**Error responses**:

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `missing_field` | Product ID is empty |
| 401 | `unauthorized` | Invalid or missing registry token |
| 500 | `delete_failed` | Index deletion failed |

```bash
curl -X DELETE -H "Authorization: Bearer $REGISTRY_TOKEN" \
  https://localhost:9090/registry/v1/products/sync/prod_abc123
```

---

## Authentication

The system uses three authentication methods depending on the API surface.

### API Key Authentication (Buyer)

Used for all buyer-facing ACE endpoints (`/ace/v1/*`).

| Header | Value |
|--------|-------|
| `X-ACE-Key` | API key created via the admin API |

```bash
curl -H "X-ACE-Key: ace_k_abcdef1234567890" https://localhost:8080/ace/v1/products
```

### Payment-as-Auth (Buyer)

An alternative to API keys where the buyer provides a payment token. The store verifies the token with the payment provider and charges per request.

| Header | Value |
|--------|-------|
| `X-ACE-Payment` | `<provider>:<token>` |

```bash
curl -H "X-ACE-Payment: stripe:tok_abc123" https://localhost:8080/ace/v1/products
```

When a paid endpoint is accessed without valid payment credentials, the server returns HTTP 402:

```json
{
  "error": "Payment required",
  "code": "payment_required",
  "pricing": {
    "price": 0.01,
    "currency": "USD",
    "accepted_providers": ["stripe", "mercadopago"],
    "details_url": "https://store.example.com/ace/v1/pricing"
  }
}
```

### Bearer Token Authentication (Admin / Registry)

Used for admin endpoints and registry operations.

| Header | Value |
|--------|-------|
| `Authorization` | `Bearer <token>` |

**ACE Store admin** -- uses the `ADMIN_TOKEN` generated at server startup:

```bash
curl -H "Authorization: Bearer $ADMIN_TOKEN" https://localhost:8080/api/v1/stores/store_001/products
```

**Registry store deletion** -- uses the `REGISTRY_ADMIN_TOKEN` environment variable:

```bash
curl -H "Authorization: Bearer $REGISTRY_ADMIN_TOKEN" -X DELETE https://localhost:9090/registry/v1/stores/store_abc123
```

**Registry product sync** -- uses the `registry_token` returned at store registration:

```bash
curl -H "Authorization: Bearer $REGISTRY_TOKEN" -X POST https://localhost:9090/registry/v1/products/sync -d '{...}'
```

---

## Response Headers

All buyer-facing ACE endpoints include pricing headers in every response.

| Header | Type | Description |
|--------|------|-------------|
| `X-ACE-Price` | string | Cost of this request in the store's currency (e.g. `"0.00"`) |
| `X-ACE-Currency` | string | ISO 4217 currency code (e.g. `"USD"`) |
| `X-ACE-Balance-Remaining` | string | Remaining prepaid balance, if applicable. Only present when the store tracks balances. |

---

## Pagination

All list endpoints return a consistent paginated envelope:

```json
{
  "data": [ ... ],
  "total": 100,
  "offset": 0,
  "limit": 20
}
```

| Field | Type | Description |
|-------|------|-------------|
| `data` | array | Array of result objects |
| `total` | int | Total number of matching items |
| `offset` | int | Current offset |
| `limit` | int | Current page size |

---

## Error Format

All errors use a standard JSON envelope:

```json
{
  "error": "Human-readable error message",
  "code": "machine_readable_code",
  "details": "Optional additional context"
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_body` | 400 | Malformed or unreadable JSON request body |
| `missing_field` | 400 | A required field is missing |
| `missing_name` | 400 | The `name` field is required |
| `invalid_quantity` | 400 | Quantity must be a positive integer |
| `invalid_pricing_model` | 400 | Per-request products cannot be added to a cart |
| `empty_cart` | 400 | Cart has no items |
| `unauthorized` | 401 | Missing or invalid authentication credentials |
| `payment_required` | 402 | Endpoint requires payment |
| `policy_denied` | 403 | Action denied by store policy |
| `not_found` | 404 | Requested resource does not exist |
| `product_not_found` | 404 | Product does not exist or is not published |
| `variant_not_found` | 404 | Variant ID does not exist on the product |
| `cart_not_found` | 404 | Cart does not exist |
| `invalid_status` | 409 | Resource is not in the required status for this operation |
| `insufficient_stock` | 409 | Not enough inventory for the requested quantity |
| `product_unavailable` | 409 | Product is no longer available |
| `invalid_store` | 400 | Could not validate the store's well-known URL |
| `search_error` | 500 | Search engine failure |
| `delete_failed` | 500 | Failed to delete from the search index |
| `server_error` | 500 | Internal server error |

---

## Data Types Reference

### Money

All monetary amounts are expressed in the smallest currency unit (cents for USD).

```json
{ "amount": 1999, "currency": "USD" }
```

This represents $19.99 USD.

### Order Status Flow

```
pending --> paid --> fulfilled
                 \-> refunded
```

### Product Status Flow

```
draft --> published --> unpublished
                   \-> published (re-publish)
```

### Payment Status Flow

```
processing --> completed
           \-> failed
```

### Health Status Values

| Value | Description |
|-------|-------------|
| `healthy` | Store's well-known URL is reachable and valid |
| `degraded` | Partial availability |
| `down` | Store is unreachable |
| `unknown` | Not yet checked |
