# ACE Connect

Connect any e-commerce platform to the ACE Protocol with one command.

## Quick Start

```bash
npx ace-connect shopify \
  --shop mystore.myshopify.com \
  --token shp_xxx \
  --registry http://localhost:8080
```

That single command fetches your Shopify catalog, starts an ACE-compatible HTTP server, and registers your store in the ACE registry so AI agents can discover, browse, and purchase your products.

## What It Does

```
+------------------+       +---------------+       +-------------------+
|                  |       |               |       |                   |
|  Shopify Store   +------>+  ace-connect  +------>+  ACE Ecosystem    |
|  (Admin API)     |       |  (HTTP server)|       |  (Registry,       |
|                  |       |               |       |   Agents, Search) |
+------------------+       +-------+-------+       +-------------------+
                                   |
                           Serves ACE endpoints:
                           /.well-known/agent-commerce
                           /ace/v1/products
                           /ace/v1/cart
                           /ace/v1/orders
```

## Shopify Setup

To connect your Shopify store you need a Custom App access token:

1. Go to your Shopify Admin panel.
2. Navigate to **Settings > Apps and sales channels > Develop apps**.
3. Click **Create an app** and give it a name (e.g. "ACE Connect").
4. Under **Configuration**, click **Configure Admin API scopes**.
5. Enable the `read_products` scope (and `read_orders` if you want order sync later).
6. Click **Save**, then go to the **API credentials** tab.
7. Click **Install app** and confirm.
8. Copy the **Admin API access token** -- this is your `--token` value.

The `--shop` value is your store's `.myshopify.com` domain (e.g. `mystore.myshopify.com`).

## CLI Options

### Common Options

| Option | Default | Description |
|--------|---------|-------------|
| `--registry <url>` | _(none)_ | ACE Registry URL. Enables discovery and cross-store search. |
| `--port <port>` | `8081` | Port for the embedded ACE HTTP server. |
| `--sync-interval <sec>` | `300` | How often to re-fetch products from the platform, in seconds. |
| `--country <code>` | `US` | Country code sent to the registry for location-based filtering. |
| `--categories <list>` | _(none)_ | Comma-separated category list for registry classification. |

### Shopify Options

| Option | Default | Description |
|--------|---------|-------------|
| `--shop <domain>` | _(required)_ | Your Shopify store domain (e.g. `mystore.myshopify.com`). |
| `--token <token>` | _(required)_ | Shopify Admin API access token. |
| `--currency <code>` | _(auto-detected)_ | Override the store currency. If omitted, detected from Shopify. |

## How It Works

1. **Fetches products from Shopify.** Uses the Shopify Admin REST API to pull all active products with pagination and automatic rate-limit retry.

2. **Serves them via ACE-compatible HTTP endpoints.** An embedded HTTP server exposes the standard ACE protocol: product catalog, cart management, order creation, and payment. The `/.well-known/agent-commerce` endpoint provides discovery metadata.

3. **Registers in the ACE registry for discovery.** If `--registry` is provided, the store registers itself and pushes its product catalog to Elasticsearch so agents can search across all connected stores.

4. **Syncs periodically to keep the catalog fresh.** A background loop re-fetches products at the configured interval and updates both the local store and the registry.

5. **Agents can browse and buy using payment-as-auth.** No API key is needed. Agents authenticate by including an `X-ACE-Payment` header (e.g. `mock:agent-123`), which simultaneously authorizes the request and initiates payment. Traditional API key auth via `X-ACE-Key` is also supported.

## Adapter Architecture

ACE Connect uses an adapter pattern to support multiple e-commerce platforms. The core interface is:

```typescript
interface EcommerceAdapter {
  name: string;
  storeName: string;
  fetchProducts(): Promise<AceProduct[]>;
}
```

Each platform adapter is a single file that implements this interface. The Shopify adapter lives at `src/adapters/shopify.ts`. Adding a new platform means writing one file that knows how to fetch products and map them to the `AceProduct` format. Everything else -- the HTTP server, sync loop, registry integration, and auth -- works unchanged.

**Current adapters:**
- Shopify

**Planned adapters:**
- Tiendanube
- WooCommerce
- CSV (import from a local file)

## Development

```bash
cd ace-connect
npm install
npm run build
node dist/index.js shopify --shop mystore.myshopify.com --token shp_xxx
```

The project uses TypeScript with zero runtime dependencies (Node.js 18+ stdlib only). The build output goes to `dist/`.

To see all available options:

```bash
node dist/index.js --help
```

## License

Apache 2.0
