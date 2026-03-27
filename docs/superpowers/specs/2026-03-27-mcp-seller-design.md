# ACE Seller MCP Server

**Date:** 2026-03-27
**Status:** Approved

## Problem

Store owners must use raw curl commands or custom scripts to manage their ACE stores via the Admin API. This is tedious for routine operations like publishing products, reviewing orders, or managing API keys. An MCP adapter lets the seller say "publish all draft products" or "show me today's orders" to Claude.

## Decision

Build a TypeScript MCP server (`mcp-seller`) that exposes all ACE Admin API operations as MCP tools, plus bulk import and registry integration. 23 tools total.

## Configuration

```json
{
  "mcpServers": {
    "ace-seller": {
      "command": "npx",
      "args": ["ace-seller-mcp"],
      "env": {
        "ACE_STORE_URL": "http://localhost:8081",
        "ACE_ADMIN_TOKEN": "your_admin_token",
        "ACE_STORE_ID": "store_demo_001",
        "ACE_REGISTRY_URL": "http://localhost:8080"
      }
    }
  }
}
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ACE_STORE_URL` | Yes | — | ACE store base URL |
| `ACE_ADMIN_TOKEN` | Yes | — | Admin API bearer token (printed at server startup) |
| `ACE_STORE_ID` | Yes | — | Store identifier (printed at server startup) |
| `ACE_REGISTRY_URL` | No | — | Registry URL. Enables `register_in_registry` and `sync_products_to_registry` tools. |

### Auth

All requests to the store use `Authorization: Bearer <ACE_ADMIN_TOKEN>`. No dual mode — the seller is always admin.

Registry requests use a `registry_token` obtained during registration, stored in `.ace-seller.json`.

## Tools (23)

### Catalog (8)

#### `list_products`

List all products in the store.

```typescript
Input: {
  offset?: number,
  limit?: number
}
Output: { data: Product[], total: number }
```

Calls: `GET /api/v1/stores/{store_id}/products`

#### `create_product`

Create a single product with variants.

```typescript
Input: {
  name: string,
  description: string,
  price: { amount: number, currency: string },
  variants?: Array<{
    name: string,
    sku?: string,
    price: { amount: number, currency: string },
    inventory: number,
    attributes?: Record<string, string>
  }>
}
Output: Product
```

Calls: `POST /api/v1/stores/{store_id}/products`

#### `bulk_create_products`

Create multiple products at once. Iterates internally, returns summary.

```typescript
Input: {
  products: Array<{
    name: string,
    description: string,
    price: { amount: number, currency: string },
    variants?: Array<{
      name: string,
      sku?: string,
      price: { amount: number, currency: string },
      inventory: number,
      attributes?: Record<string, string>
    }>
  }>
}
Output: {
  created: number,
  errors: Array<{ index: number, name: string, error: string }>
}
```

Calls `POST /api/v1/stores/{store_id}/products` for each product. Collects successes and failures.

#### `update_product`

Update a product's fields.

```typescript
Input: {
  product_id: string,
  name?: string,
  description?: string,
  price?: { amount: number, currency: string }
}
Output: Product
```

Calls: `PATCH /api/v1/stores/{store_id}/products/{id}`

#### `delete_product`

Delete a product.

```typescript
Input: { product_id: string }
Output: { deleted: true }
```

Calls: `DELETE /api/v1/stores/{store_id}/products/{id}`

#### `publish_product`

Make a product visible to buyers.

```typescript
Input: { product_id: string }
Output: Product
```

Calls: `POST /api/v1/stores/{store_id}/products/{id}/publish`

#### `unpublish_product`

Hide a product from buyers.

```typescript
Input: { product_id: string }
Output: Product
```

Calls: `POST /api/v1/stores/{store_id}/products/{id}/unpublish`

#### `update_inventory`

Update stock for a variant.

```typescript
Input: {
  variant_id: string,
  inventory: number
}
Output: { updated: true }
```

Calls: `PATCH /api/v1/stores/{store_id}/variants/{id}/inventory`

### Orders (4)

#### `list_orders`

List all orders.

```typescript
Input: { offset?: number, limit?: number }
Output: { data: Order[], total: number }
```

Calls: `GET /api/v1/stores/{store_id}/orders`

#### `get_order`

Get order details.

```typescript
Input: { order_id: string }
Output: Order
```

Calls: `GET /api/v1/stores/{store_id}/orders/{id}`

#### `fulfill_order`

Mark an order as fulfilled/shipped.

```typescript
Input: { order_id: string }
Output: Order
```

Calls: `POST /api/v1/stores/{store_id}/orders/{id}/fulfill`

#### `refund_order`

Refund an order. May require approval depending on store policies.

```typescript
Input: { order_id: string }
Output: Order
```

Calls: `POST /api/v1/stores/{store_id}/orders/{id}/refund`

### Policies (5)

#### `get_policies`

Get current store policies.

```typescript
Input: {}
Output: Policy[]
```

Calls: `GET /api/v1/stores/{store_id}/policies`

#### `update_policies`

Replace all store policies.

```typescript
Input: {
  policies: Array<{
    action: string,
    effect: "allow" | "deny" | "approval"
  }>
}
Output: Policy[]
```

Calls: `PUT /api/v1/stores/{store_id}/policies`

#### `list_approvals`

List pending approval requests.

```typescript
Input: {}
Output: Approval[]
```

Calls: `GET /api/v1/stores/{store_id}/approvals`

#### `approve_action`

Approve a pending action.

```typescript
Input: { approval_id: string }
Output: Approval
```

Calls: `POST /api/v1/stores/{store_id}/approvals/{id}/approve`

#### `reject_action`

Reject a pending action.

```typescript
Input: { approval_id: string }
Output: Approval
```

Calls: `POST /api/v1/stores/{store_id}/approvals/{id}/reject`

### Security (4)

#### `list_api_keys`

List all API keys issued for buyer agents.

```typescript
Input: {}
Output: APIKey[]
```

Calls: `GET /api/v1/stores/{store_id}/api-keys`

#### `create_api_key`

Create a new API key for a buyer agent.

```typescript
Input: {
  name: string,
  scopes: string[]  // e.g., ["catalog:read", "cart:write", "orders:write", "payments:write"]
}
Output: { key: string, ...APIKey }  // key shown only once
```

Calls: `POST /api/v1/stores/{store_id}/api-keys`

#### `delete_api_key`

Revoke an API key.

```typescript
Input: { key_id: string }
Output: { deleted: true }
```

Calls: `DELETE /api/v1/stores/{store_id}/api-keys/{id}`

#### `list_audit_logs`

View the audit trail of all actions on the store.

```typescript
Input: {
  action?: string,
  actor?: string,
  offset?: number,
  limit?: number
}
Output: { data: AuditEntry[], total: number }
```

Calls: `GET /api/v1/stores/{store_id}/audit-logs`

### Registry (2)

Only available when `ACE_REGISTRY_URL` is configured.

#### `register_in_registry`

Register the store in the ACE registry for discovery. Persists the registry_token to `.ace-seller.json`.

```typescript
Input: {
  categories?: string[],
  country?: string
}
Output: { store_id: string, registered: true }
```

Calls: `POST /registry/v1/stores` with the store's `.well-known/agent-commerce` URL. If `.ace-seller.json` exists with a valid token, skips registration and returns the existing ID.

#### `sync_products_to_registry`

Push all published products to the registry search index.

```typescript
Input: {}
Output: { synced: number, errors: number }
```

Flow:
1. Fetch all products via `GET /api/v1/stores/{store_id}/products`
2. Filter to `status === "published"`
3. Map each to `ProductSyncRequest` (with price_range, variants_summary, in_stock, location)
4. Push via `POST /registry/v1/products/sync` with the registry_token

## Project Structure

```
mcp-seller/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts              # MCP server entry point, tool registration
│   ├── config.ts             # Env var parsing and validation
│   ├── client.ts             # HTTP client with admin Bearer auth
│   └── tools/
│       ├── catalog.ts        # 8 tools: list, create, bulk_create, update, delete, publish, unpublish, inventory
│       ├── orders.ts         # 4 tools: list, get, fulfill, refund
│       ├── policies.ts       # 5 tools: get, update, list_approvals, approve, reject
│       ├── security.ts       # 4 tools: list_keys, create_key, delete_key, audit_logs
│       └── registry.ts       # 2 tools: register, sync
└── README.md
```

## Dependencies

- `@modelcontextprotocol/sdk` — Official MCP SDK
- `zod` — Schema validation
- No other runtime dependencies. HTTP via native `fetch`.

## Persistence

`.ace-seller.json` stores the registry token to avoid re-registration across restarts:

```json
{
  "store_url": "http://localhost:8081",
  "store_id": "str_abc123",
  "registry_token": "rgt_xxx...",
  "registered_at": "2026-03-27T..."
}
```

## What This Does NOT Include

- No buyer operations (use mcp-buyer for that)
- No ace-connect management
- No real-time notifications (polling only)
- No multi-store management (one MCP instance per store)
