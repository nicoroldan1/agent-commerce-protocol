# ACE Buyer MCP Server

MCP server that lets AI agents discover stores, browse catalogs, and make purchases through the ACE Protocol.

## Quick Start

Add this to your Claude Desktop configuration (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "ace-buyer": {
      "command": "npx",
      "args": ["ace-buyer-mcp"],
      "env": {
        "ACE_STORE_URL": "http://localhost:8081",
        "ACE_PAYMENT_PROVIDER": "mock"
      }
    }
  }
}
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ACE_REGISTRY_URL` | No | -- | URL of the ACE Registry for multi-store discovery |
| `ACE_STORE_URL` | No | -- | URL of a single ACE store (used in single-store mode) |
| `ACE_API_KEY` | No | -- | API key sent as `Authorization: Bearer` header to stores |
| `ACE_PAYMENT_PROVIDER` | No | `mock` | Payment provider to use when paying for orders |
| `ACE_PAYMENT_TOKEN` | No | -- | Token passed to the payment provider |

At least one of `ACE_REGISTRY_URL` or `ACE_STORE_URL` must be set.

## Two Modes

### Single-Store Mode

Set only `ACE_STORE_URL`. The server connects directly to one store. Discovery tools (`discover_stores`, `search_products`) are not available. All catalog and purchase tools target the configured store.

```bash
ACE_STORE_URL=http://localhost:8081 node dist/index.js
```

### Registry Mode

Set `ACE_REGISTRY_URL` (and optionally `ACE_STORE_URL` as a default). All 13 tools are available. Discovery tools query the registry, and catalog/purchase tools can target any store by passing `store_url` in the request.

```bash
ACE_REGISTRY_URL=http://localhost:8080 ACE_STORE_URL=http://localhost:8081 node dist/index.js
```

## Tools Reference

### Discovery

| Tool | Description |
|------|-------------|
| `discover_stores` | Search for ACE stores in the registry by name, category, or country |
| `search_products` | Search for products across all stores in the registry |

### Catalog

| Tool | Description |
|------|-------------|
| `browse_store` | List products from an ACE store with optional filters |
| `get_product` | Get full product details including variants and pricing |
| `get_pricing` | Get a store's pricing schedule for all endpoints |

### Purchase

| Tool | Description |
|------|-------------|
| `create_cart` | Create a new shopping cart at an ACE store |
| `get_cart` | Get the current state of a shopping cart |
| `add_to_cart` | Add a product to a shopping cart |
| `shipping_quote` | Get shipping options and prices for a destination |
| `place_order` | Convert a cart into an order |
| `pay_order` | Pay for an order using the configured payment provider |
| `get_order` | Get full order details including items and payment status |
| `payment_status` | Check payment status for an order |

## Example Conversation

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

## Development

```bash
cd mcp-buyer
npm install
npm run build
ACE_STORE_URL=http://localhost:8081 node dist/index.js
```

The server communicates over stdio using the MCP protocol. It logs startup information to stderr so it does not interfere with the JSON-RPC message stream on stdout.

## License

Apache 2.0
