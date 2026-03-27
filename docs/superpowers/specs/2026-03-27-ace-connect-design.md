# ACE Connect — Platform Connectors

**Date:** 2026-03-27
**Status:** Approved

## Problem

Sellers who already have stores on platforms like Shopify, Tiendanube, or WooCommerce cannot easily expose their catalogs to the ACE ecosystem. They would need to manually recreate their products via the ACE Admin API. This is a non-starter for adoption.

## Decision

Build `ace-connect`, a CLI tool that connects any e-commerce platform to ACE with a single command. It uses an adapter pattern so adding new platforms requires implementing one interface. Shopify is the first adapter.

```bash
npx ace-connect shopify \
  --shop mystore.myshopify.com \
  --token shp_xxx \
  --registry http://localhost:8080
```

One command. Zero config files. The store is live on ACE.

## Architecture

```
E-commerce Platform              ACE Connect                         ACE Ecosystem
───────────────────              ───────────                         ─────────────
                                 ┌─────────────────────┐
Shopify API    ──── adapter ──►  │  EcommerceAdapter    │
Tiendanube API ──── adapter ──►  │  ↓                   │
CSV file       ──── adapter ──►  │  In-memory catalog   │
                   (periodic)    │                      │
                                 │  HTTP Server         │
                                 │  ├ /.well-known/     │ ◄──── Buyer agents
                                 │  ├ /ace/v1/products  │
                                 │  ├ /ace/v1/cart      │
                                 │  ├ /ace/v1/orders    │
                                 │  └ /ace/v1/pricing   │
                                 │                      │
                                 │  Registry sync       │ ────► ACE Registry
                                 └─────────────────────┘        (Elasticsearch)
```

## Adapter Interface

The core abstraction. Each e-commerce platform implements this:

```typescript
interface EcommerceAdapter {
  /** Adapter identifier (e.g., "shopify", "tiendanube", "csv") */
  name: string;

  /** Human-readable store name for registry */
  storeName: string;

  /** Fetch all products from the platform and return as ACE products */
  fetchProducts(): Promise<AceProduct[]>;
}

interface AceProduct {
  id: string;
  name: string;
  description: string;
  category: string;
  tags: string[];
  price: { amount: number; currency: string };
  variants: AceVariant[];
  imageUrl: string;
  status: "published" | "draft";
  pricingModel: "fixed";           // Always "fixed" for platform imports
  pricePerRequest: 0;              // Not applicable for platform products
  createdAt: string;               // ISO 8601, set from sync time
  updatedAt: string;               // ISO 8601, updated on each sync
}

interface AceVariant {
  id: string;
  name: string;
  sku: string;
  price: { amount: number; currency: string };
  inventory: number;
  attributes: Record<string, string>;  // e.g., { "color": "Red", "size": "L" }
}
```

Adding a new platform = one file implementing `EcommerceAdapter`. Zero changes to server, sync, or registry code.

## Shopify Adapter

### Authentication

The seller creates a Custom App in Shopify Admin:
1. Admin → Settings → Apps and sales channels → Develop apps
2. Create an app → Configure Admin API scopes: `read_products`
3. Install app → Copy the Admin API access token

The token is passed via `--token` flag.

### API Usage

Uses Shopify REST Admin API (no GraphQL, no SDK):

```
GET https://{shop}/admin/api/2025-01/products.json?status=active&limit=250
```

API version `2025-01` (stable as of early 2026). Pagination via `Link` header for stores with 250+ products.

**Rate limiting:** Shopify uses a leaky bucket (40 requests/app, refills at 2/sec). On 429 responses, retry with exponential backoff (1s, 2s, 4s, max 3 retries). For the initial sync of large catalogs (1000+ products = 4+ pages), this adds a few seconds of delay.

**Product IDs:** Shopify numeric IDs (e.g., `7654321098`) are used directly as strings. Prefixed with `shp_` for namespacing: `shp_7654321098`.

**Currency:** Fetched from `GET /admin/api/2025-01/shop.json` on startup (field: `currency`). Cached for the session. Fallback: `--currency` CLI flag or `USD` default.

### Data Mapping

| Shopify field | ACE field |
|---------------|-----------|
| `product.title` | `name` |
| `product.body_html` (tags stripped) | `description` |
| `product.product_type` | `category` |
| `product.tags` (comma-split) | `tags` |
| `product.images[0].src` | `imageUrl` |
| `product.status == "active"` | `status: "published"` |
| `variant.title` | `variant.name` |
| `variant.sku` | `variant.sku` |
| `variant.price` (string → cents) | `variant.price` |
| `variant.inventory_quantity` | `variant.inventory` |
| `product.variants[0].price` | `price` (product-level, first variant) |
| `variant.option1/option2/option3` | `variant.attributes` (e.g., `{"Size": "L", "Color": "Red"}`) |

### Price Conversion

Shopify prices are strings like `"79.99"`. Convert to ACE cents: `Math.round(parseFloat(price) * 100)`. Currency from Shopify store settings (defaults to USD).

## CLI Interface

```bash
npx ace-connect <platform> [options]

Platforms:
  shopify       Connect a Shopify store

Common options:
  --registry <url>       ACE Registry URL (enables discovery + product search)
  --port <port>          Local server port (default: 8081)
  --sync-interval <sec>  Sync interval in seconds (default: 300)
  --country <code>       Store country for registry (default: US)
  --categories <list>    Comma-separated categories for registry

Shopify options:
  --shop <domain>        Shopify store domain (required)
  --token <token>        Shopify Admin API access token (required)
  --currency <code>      Override currency (auto-detected from Shopify if omitted)
```

### Startup Flow

1. Parse CLI args, validate required params
2. Create the appropriate adapter (ShopifyAdapter, etc.)
3. Initial sync: `adapter.fetchProducts()` → populate in-memory store
4. Start HTTP server on `--port`
5. If `--registry`: register store + push products to registry index
6. Start periodic sync loop every `--sync-interval` seconds
7. Register SIGINT/SIGTERM handlers for graceful shutdown (log + stop sync loop + close server)
8. Log: `"ACE Connect running on :8081 — 47 products from Shopify (mystore.myshopify.com)"`

## Embedded ACE Server

The connector embeds a minimal ACE-compatible HTTP server using Node.js `node:http`.

### Endpoints

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /.well-known/agent-commerce` | None | Discovery with payment_auth: mock |
| `GET /ace/v1/products` | Dual (key or payment) | List products with filters, pagination |
| `GET /ace/v1/products/{id}` | Dual | Product detail with variants |
| `GET /ace/v1/pricing` | None | Pricing schedule (all free) |
| `POST /ace/v1/cart` | Dual | Create cart |
| `POST /ace/v1/cart/{id}/items` | Dual | Add item to cart |
| `GET /ace/v1/cart/{id}` | Dual | Get cart |
| `POST /ace/v1/orders` | Dual | Create order from cart |
| `POST /ace/v1/orders/{id}/pay` | Dual | Pay order (mock) |
| `GET /ace/v1/orders/{id}` | Dual | Get order |
| `GET /ace/v1/orders/{id}/pay/status` | Dual | Payment status |
| `POST /ace/v1/shipping/quote` | Dual | Mock shipping options (standard, express, overnight) |

The `.well-known` response advertises capabilities: `["catalog", "cart", "orders", "payments", "shipping"]`. The shipping endpoint returns mock options (not connected to Shopify shipping).

### Auth

Accepts `X-ACE-Payment: mock:*` by default (payment-as-auth, anyone can access). No API key setup required. The goal is zero friction for the seller.

### Error Responses

The embedded server uses the same error format as the real ACE server for protocol compatibility:

```json
{ "error": "message", "code": "error_code", "details": "optional" }
```

### Orders

Orders are stored in-memory only. They do NOT create orders in Shopify (that requires Shopify Checkout API or Draft Orders API, which is significantly more complex). This is a known limitation for v1. The order flow validates the cart, decrements in-memory inventory, and records the order — enough for agents to complete the full purchase flow.

## In-Memory Store

```typescript
class InMemoryStore {
  products: Map<string, AceProduct>;
  carts: Map<string, Cart>;
  orders: Map<string, Order>;

  replaceProducts(products: AceProduct[]): void;  // Called on each sync
  // ... cart/order CRUD methods matching ace-server patterns
}
```

`replaceProducts` atomically swaps the product catalog on each sync cycle. Existing carts/orders retain their snapshot prices — if a product price changes mid-cart, the cart keeps the old price. This is intentional (matches real e-commerce behavior where cart prices are locked at add time).

## Registry Integration

When `--registry` is provided:

1. **On startup:** Check for `.ace-connect.json` in the current directory. If it exists and contains a valid `registry_token` + `store_id` for this shop, reuse them. Otherwise, register via `POST /registry/v1/stores` → receive `registry_token` → persist to `.ace-connect.json`.
2. **On each sync:** Push products to registry via `POST /registry/v1/products/sync` using the registry token.
3. **Product sync payload:** Maps each AceProduct to a `ProductSyncRequest` with price_range, variants_summary, rating (default 0), location (from --country).

### Persistence File (`.ace-connect.json`)

```json
{
  "shop": "mystore.myshopify.com",
  "store_id": "str_abc123",
  "registry_token": "rgt_xxx...",
  "registered_at": "2026-03-27T..."
}
```

This prevents duplicate store registrations across restarts. The file is scoped by shop domain — if the domain changes, a new registration is created.

## Project Structure

```
ace-connect/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts              # CLI entry point, arg parsing, orchestration
│   ├── adapters/
│   │   ├── adapter.ts        # EcommerceAdapter interface + AceProduct types
│   │   └── shopify.ts        # Shopify adapter implementation
│   ├── server/
│   │   ├── app.ts            # HTTP server with ACE endpoints
│   │   ├── store.ts          # In-memory store (products, carts, orders)
│   │   └── auth.ts           # Dual auth (mock payment-as-auth)
│   ├── sync.ts               # Periodic sync using adapter.fetchProducts()
│   └── registry.ts           # Registry registration + product push
└── README.md
```

## Dependencies

- Zero runtime dependencies beyond Node.js stdlib
- `node:http` for the server
- `node:crypto` for ID generation
- `fetch` (global in Node 18+) for Shopify API and registry calls
- No Express, no Fastify, no Shopify SDK

## Dev Dependencies

- `typescript`
- `@types/node`

## What This Does NOT Include

- No real orders in Shopify (mock orders only)
- No real payments
- No Shopify webhooks (polling only)
- No OAuth flow (manual token)
- No state persistence (in-memory, re-syncs on restart)
- No other platform adapters yet (architecture ready, implementation deferred)

## Future Adapters (Not in Scope)

These follow the same pattern — one file implementing `EcommerceAdapter`:

- `tiendanube.ts` — Tiendanube REST API
- `woocommerce.ts` — WooCommerce REST API
- `csv.ts` — Import from CSV/JSON file
- `manual.ts` — Interactive product entry via CLI prompts
