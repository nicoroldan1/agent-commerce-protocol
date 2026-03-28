# ACE Seller MCP Server

MCP server for ACE store owners -- manage catalog, orders, policies, and API keys through Claude.

## Quick Start

Add this to your Claude Desktop configuration (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "ace-seller": {
      "command": "node",
      "args": ["/path/to/mcp-seller/dist/index.js"],
      "env": {
        "ACE_STORE_URL": "http://localhost:8081",
        "ACE_ADMIN_TOKEN": "your_admin_token_from_server_startup",
        "ACE_STORE_ID": "store_demo_001",
        "ACE_REGISTRY_URL": "http://localhost:8080"
      }
    }
  }
}
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ACE_STORE_URL` | Yes | -- | URL of the ACE store to manage |
| `ACE_ADMIN_TOKEN` | Yes | -- | Admin Bearer token (printed at server startup) |
| `ACE_STORE_ID` | Yes | -- | Store identifier used in Admin API paths |
| `ACE_REGISTRY_URL` | No | -- | URL of the ACE Registry (enables registry tools) |

## Tools Reference

### Catalog (8 tools)

| Tool | Description |
|------|-------------|
| `list_products` | List all products in the store with optional pagination |
| `create_product` | Create a new product with name, description, price, and variants |
| `bulk_create_products` | Create multiple products in a single operation |
| `update_product` | Update an existing product's name, description, or price |
| `delete_product` | Delete a product from the catalog |
| `publish_product` | Publish a draft product so it becomes visible to buyers |
| `unpublish_product` | Unpublish a product, hiding it from buyers |
| `update_inventory` | Update the inventory count for a specific variant |

### Orders (4 tools)

| Tool | Description |
|------|-------------|
| `list_orders` | List all orders with optional pagination |
| `get_order` | Get full details for a specific order |
| `fulfill_order` | Mark an order as fulfilled |
| `refund_order` | Initiate a refund for an order |

### Policies (5 tools)

| Tool | Description |
|------|-------------|
| `get_policies` | Get the current policy configuration for the store |
| `update_policies` | Update policy rules (allow, deny, or approval per action) |
| `list_approvals` | List pending approval requests |
| `approve_action` | Approve a pending action |
| `reject_action` | Reject a pending action |

### Security (4 tools)

| Tool | Description |
|------|-------------|
| `list_api_keys` | List all API keys issued for the store |
| `create_api_key` | Create a new API key with specific scopes |
| `delete_api_key` | Revoke and delete an API key |
| `list_audit_logs` | Query the immutable audit log with optional filters |

### Registry (2 tools)

| Tool | Description |
|------|-------------|
| `register_in_registry` | Register the store in the ACE Registry for discovery |
| `sync_products_to_registry` | Sync all published products to the registry search index |

These tools are only available when `ACE_REGISTRY_URL` is configured.

## Example Conversation

```
User: Show me all my products.
Claude: [calls list_products] You have 7 products: ...

User: Create a new product called "Wireless Mouse" at $29.99 with variants Small and Large.
Claude: [calls create_product] Created "Wireless Mouse" (prod_8). It's in draft status.

User: Publish it.
Claude: [calls publish_product] "Wireless Mouse" is now published and visible to buyers.

User: Sync everything to the registry.
Claude: [calls sync_products_to_registry] Synced 8 products to the registry search index.
```

## Development

```bash
cd mcp-seller
npm install
npm run build
ACE_STORE_URL=http://localhost:8081 ACE_ADMIN_TOKEN=your_token ACE_STORE_ID=store_demo_001 node dist/index.js
```

The server communicates over stdio using the MCP protocol. It logs startup information to stderr so it does not interfere with the JSON-RPC message stream on stdout.

## License

Apache 2.0
