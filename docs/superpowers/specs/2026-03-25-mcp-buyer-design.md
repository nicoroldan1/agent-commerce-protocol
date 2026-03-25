# ACE Buyer MCP Server

**Date:** 2026-03-25
**Status:** Approved

## Problem

Using the ACE protocol today requires making 7+ raw HTTP calls to discover stores, browse catalogs, and complete purchases. This is too complex for non-technical users and even for AI agents that lack custom HTTP client code. An MCP adapter would let any MCP-compatible agent (Claude, GPT, etc.) interact with ACE stores through natural language.

## Decision

Build a TypeScript MCP server (`mcp-buyer`) that exposes ACE buyer operations as MCP tools. It supports two modes: single-store (direct URL) and registry (discovery + search across all stores). Both modes can be active simultaneously.

## Configuration

Users configure the MCP server in their agent's config (e.g., `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "ace-buyer": {
      "command": "npx",
      "args": ["ace-buyer-mcp"],
      "env": {
        "ACE_REGISTRY_URL": "http://localhost:8080",
        "ACE_STORE_URL": "http://localhost:8081",
        "ACE_API_KEY": "",
        "ACE_PAYMENT_PROVIDER": "mock",
        "ACE_PAYMENT_TOKEN": ""
      }
    }
  }
}
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ACE_REGISTRY_URL` | No | — | Registry URL. Enables `discover_stores` and `search_products` tools. |
| `ACE_STORE_URL` | No | — | Direct store URL. Used as default when `store_url` is not passed to tools. |
| `ACE_API_KEY` | No | — | API key for store auth. Sent as `X-ACE-Key` header. |
| `ACE_PAYMENT_PROVIDER` | No | `mock` | Payment provider for payment-as-auth. |
| `ACE_PAYMENT_TOKEN` | No | — | Payment token. If empty with `mock` provider, auto-generates a session token. |

At least one of `ACE_REGISTRY_URL` or `ACE_STORE_URL` must be set. If neither is set, the server fails at startup with a clear error message.

### Auth Strategy

The MCP server adds auth headers to every store request automatically:

1. If `ACE_API_KEY` is set: `X-ACE-Key: <key>`
2. Else if `ACE_PAYMENT_PROVIDER` is set: `X-ACE-Payment: <provider>:<token>`
3. Else if provider is `mock` and no token: `X-ACE-Payment: mock:mcp_session_<random>`

API key takes priority over payment token (matching ACE server behavior).

## Tools

### Discovery & Search

#### `discover_stores`

Search for stores in the registry. Only available when `ACE_REGISTRY_URL` is configured.

```typescript
Input: {
  query?: string,      // Text search on store name
  category?: string,   // Filter by category
  country?: string,    // Filter by country
  currency?: string,   // Filter by currency
  offset?: number,     // Pagination offset (default 0)
  limit?: number       // Pagination limit (default 20, max 100)
}

Output: {
  stores: Array<{
    id: string,
    name: string,
    well_known_url: string,
    categories: string[],
    country: string,
    currencies: string[],
    capabilities: string[],
    health_status: string
  }>,
  total: number
}
```

To use a discovered store in other tools, pass its `id` to store URL resolution (the MCP fetches the store's `.well-known/agent-commerce` to get the `ace_base_url`).

#### `search_products`

Search for products across all stores via the registry's Elasticsearch index. Only available when `ACE_REGISTRY_URL` is configured.

```typescript
Input: {
  query: string,         // Full-text search
  category?: string,     // Filter by category
  country?: string,      // Filter by country
  currency?: string,     // Filter by currency
  price_min?: number,    // Min price in USD
  price_max?: number,    // Max price in USD
  in_stock?: boolean,    // Filter by stock (default true)
  sort?: string,         // "relevance" | "price_asc" | "price_desc" | "rating"
  offset?: number,       // Pagination offset (default 0)
  limit?: number         // Pagination limit (default 20, max 100)
}

Output: {
  products: Array<{
    product_id: string,
    store_id: string,
    store_name: string,
    name: string,
    description: string,
    category: string,
    price_range: { min: number, max: number, currency: string },
    variants_summary: string[],
    in_stock: boolean,
    rating: { average: number, count: number },
    location: { country: string, region: string }
  }>,
  total: number
}
```

### Catalog

#### `browse_store`

List products from a specific store.

```typescript
Input: {
  store_url?: string,    // Store base URL (uses ACE_STORE_URL if omitted)
  category?: string,
  query?: string,
  offset?: number,       // Pagination offset (default 0)
  limit?: number         // Pagination limit (default 20, max 100)
}

Output: {
  products: Array<Product>,
  total: number
}
```

#### `get_product`

Get full product details including variants.

```typescript
Input: {
  store_url?: string,
  product_id: string
}

Output: Product  // Full product with variants, pricing, stock
```

#### `get_pricing`

Get a store's pricing schedule.

```typescript
Input: {
  store_url?: string
}

Output: {
  default_currency: string,
  endpoints: Array<{ method: string, path: string, price: number }>
}
```

### Purchase

#### `create_cart`

Create a new shopping cart at a store.

```typescript
Input: {
  store_url?: string
}

Output: Cart  // Full cart object: id, items (empty), total, created_at, updated_at
```

#### `get_cart`

Get the current state of a cart.

```typescript
Input: {
  store_url?: string,
  cart_id: string
}

Output: Cart  // Full cart: id, items, total, created_at, updated_at
```

#### `add_to_cart`

Add a product to an existing cart.

```typescript
Input: {
  store_url?: string,
  cart_id: string,
  product_id: string,
  quantity: number,
  variant_id?: string
}

Output: Cart  // Updated cart with items and total
```

#### `shipping_quote`

Get shipping options for items to a destination.

```typescript
Input: {
  store_url?: string,
  items: Array<{ product_id: string, quantity: number }>,
  destination: { country: string, state?: string, city?: string, postal_code?: string }
}

Output: {
  options: Array<{
    id: string,
    name: string,
    price: { amount: number, currency: string },  // amount in cents
    estimated_days: number
  }>
}
```

#### `place_order`

Convert a cart into an order.

```typescript
Input: {
  store_url?: string,
  cart_id: string
}

Output: Order  // Full order: id, cart_id, items, total, status, created_at
```

#### `pay_order`

Pay for an order.

```typescript
Input: {
  store_url?: string,
  order_id: string,
  provider?: string  // Payment provider override (default: ACE_PAYMENT_PROVIDER env var)
}

Output: Payment  // Full payment: id, order_id, status, provider, amount, payment_url
```

The `provider` parameter controls which payment provider to use for this specific payment. If omitted, uses `ACE_PAYMENT_PROVIDER` from config. This is distinct from auth — the auth header is always injected automatically.

#### `get_order`

Get full order details.

```typescript
Input: {
  store_url?: string,
  order_id: string
}

Output: Order  // Full order: id, cart_id, items, total, status, payment, created_at
```

#### `payment_status`

Check payment status for an order.

```typescript
Input: {
  store_url?: string,
  order_id: string
}

Output: Payment  // Payment: id, order_id, status, provider, amount
```

### Money Amounts

All `Money` / `amount` fields from the ACE API are in **cents** (smallest currency unit). The MCP passes them through as-is. For example, `{ amount: 7999, currency: "USD" }` means $79.99. The `price_per_request` and `X-ACE-Price` header values are in USD with decimals (e.g., `0.003`).

## Error Handling

All tools return structured errors that include the ACE error code when available:

- **Store unreachable:** `"Failed to connect to store at <url>: <error>"`
- **Registry unreachable:** `"Failed to connect to registry at <url>: <error>"`
- **HTTP 401 (Unauthorized):** `"Authentication failed: <ace_error_message>. Check ACE_API_KEY or ACE_PAYMENT_* config."`
- **HTTP 402 (Payment Required):** `"Payment required. Price: $<price> <currency>. Accepted providers: <list>. Configure ACE_PAYMENT_PROVIDER."`
- **HTTP 4xx/5xx:** `"ACE API error (<status>): <ace_error_message> [code: <ace_code>]"`

Errors are returned as MCP tool errors (not thrown exceptions), so the agent can reason about them and retry or ask the user for input.

## Store URL Resolution

When a tool receives no `store_url`:

1. If `ACE_STORE_URL` is configured, use it
2. Otherwise, return an error: `"No store URL provided. Set ACE_STORE_URL or pass store_url parameter. Use discover_stores to find stores."`

When the user uses `search_products` and picks a product, the result includes `store_id`. The MCP resolves the store's base URL by:

1. Looking up the store in the registry: `GET /registry/v1/stores/{store_id}`
2. Fetching the store's `.well-known/agent-commerce` from the `well_known_url`
3. Using the `ace_base_url` from the response

This resolution is cached in memory for the session (store_id → ace_base_url).

## Project Structure

```
mcp-buyer/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts              # MCP server entry point, tool registration
│   ├── config.ts             # Env var parsing and validation
│   ├── tools/
│   │   ├── discovery.ts      # discover_stores, search_products
│   │   ├── catalog.ts        # browse_store, get_product, get_pricing
│   │   └── purchase.ts       # create_cart, add_to_cart, place_order, pay_order, order_status
│   └── client/
│       ├── registry.ts       # HTTP client for registry API
│       └── store.ts          # HTTP client for ACE store API (with auth header injection)
└── README.md                 # Setup instructions for end users
```

### Dependencies

- `@modelcontextprotocol/sdk` — Official MCP SDK
- `zod` — Schema validation for tool inputs (required by MCP SDK)
- No other runtime dependencies. HTTP via native `fetch`.

### Build & Publish

- Built with `tsc` to `dist/`
- Entry point: `dist/index.js`
- Published to npm as `ace-buyer-mcp` for `npx` usage
- Also runnable locally: `node dist/index.js`

## Go Workspace Integration

The `mcp-buyer/` directory is a standalone TypeScript project inside the existing Go monorepo. It does NOT affect the Go workspace (`go.work`). It has its own `package.json` and `tsconfig.json`.

## What This Does NOT Include

- No state persistence between sessions (no database)
- No autonomous purchase decisions (the agent decides, MCP executes)
- No wallet or balance management
- No seller MCP (separate scope, Phase 2b)
- No npm publishing automation (manual for now)
