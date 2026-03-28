#!/usr/bin/env node

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { loadConfig } from "./config.js";
import { AdminClient } from "./client.js";
import {
  createCatalogTools,
  listProductsSchema,
  createProductSchema,
  bulkCreateProductsSchema,
  updateProductSchema,
  deleteProductSchema,
  publishProductSchema,
  unpublishProductSchema,
  updateInventorySchema,
} from "./tools/catalog.js";
import {
  createOrderTools,
  listOrdersSchema,
  getOrderSchema,
  fulfillOrderSchema,
  refundOrderSchema,
} from "./tools/orders.js";
import {
  createPolicyTools,
  getPoliciesSchema,
  updatePoliciesSchema,
  listApprovalsSchema,
  approveActionSchema,
  rejectActionSchema,
} from "./tools/policies.js";
import {
  createSecurityTools,
  listApiKeysSchema,
  createApiKeySchema,
  deleteApiKeySchema,
  listAuditLogsSchema,
} from "./tools/security.js";
import {
  createRegistryTools,
  registerInRegistrySchema,
  syncProductsToRegistrySchema,
} from "./tools/registry.js";

const config = loadConfig();
const client = new AdminClient(config);

const catalog = createCatalogTools(client, config);
const orders = createOrderTools(client, config);
const policies = createPolicyTools(client, config);
const security = createSecurityTools(client, config);
const registry = config.registryUrl ? createRegistryTools(client, config) : null;

const server = new McpServer({
  name: "ace-seller",
  version: "0.1.0",
});

// Helper to wrap tool handlers with error handling
function toolHandler(fn: (args: any) => Promise<any>) {
  return async (args: any) => {
    try {
      const result = await fn(args);
      return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
    } catch (error: any) {
      return {
        content: [{ type: "text" as const, text: `Error: ${error.message}` }],
        isError: true,
      };
    }
  };
}

// Catalog tools (8)
server.tool(
  "list_products",
  "List all products in the store",
  listProductsSchema.shape,
  toolHandler(catalog.list_products)
);

server.tool(
  "create_product",
  "Create a new product with variants",
  createProductSchema.shape,
  toolHandler(catalog.create_product)
);

server.tool(
  "bulk_create_products",
  "Create multiple products at once",
  bulkCreateProductsSchema.shape,
  toolHandler(catalog.bulk_create_products)
);

server.tool(
  "update_product",
  "Update a product's name, description, or price",
  updateProductSchema.shape,
  toolHandler(catalog.update_product)
);

server.tool(
  "delete_product",
  "Delete a product",
  deleteProductSchema.shape,
  toolHandler(catalog.delete_product)
);

server.tool(
  "publish_product",
  "Publish a product (make visible to buyers)",
  publishProductSchema.shape,
  toolHandler(catalog.publish_product)
);

server.tool(
  "unpublish_product",
  "Unpublish a product (hide from buyers)",
  unpublishProductSchema.shape,
  toolHandler(catalog.unpublish_product)
);

server.tool(
  "update_inventory",
  "Update inventory for a variant",
  updateInventorySchema.shape,
  toolHandler(catalog.update_inventory)
);

// Order tools (4)
server.tool(
  "list_orders",
  "List all orders",
  listOrdersSchema.shape,
  toolHandler(orders.list_orders)
);

server.tool(
  "get_order",
  "Get order details",
  getOrderSchema.shape,
  toolHandler(orders.get_order)
);

server.tool(
  "fulfill_order",
  "Mark an order as fulfilled/shipped",
  fulfillOrderSchema.shape,
  toolHandler(orders.fulfill_order)
);

server.tool(
  "refund_order",
  "Refund an order",
  refundOrderSchema.shape,
  toolHandler(orders.refund_order)
);

// Policy tools (5)
server.tool(
  "get_policies",
  "Get store policies",
  getPoliciesSchema.shape,
  toolHandler(policies.get_policies)
);

server.tool(
  "update_policies",
  "Update store policies",
  updatePoliciesSchema.shape,
  toolHandler(policies.update_policies)
);

server.tool(
  "list_approvals",
  "List pending approval requests",
  listApprovalsSchema.shape,
  toolHandler(policies.list_approvals)
);

server.tool(
  "approve_action",
  "Approve a pending action",
  approveActionSchema.shape,
  toolHandler(policies.approve_action)
);

server.tool(
  "reject_action",
  "Reject a pending action",
  rejectActionSchema.shape,
  toolHandler(policies.reject_action)
);

// Security tools (4)
server.tool(
  "list_api_keys",
  "List all API keys",
  listApiKeysSchema.shape,
  toolHandler(security.list_api_keys)
);

server.tool(
  "create_api_key",
  "Create an API key for a buyer agent",
  createApiKeySchema.shape,
  toolHandler(security.create_api_key)
);

server.tool(
  "delete_api_key",
  "Revoke an API key",
  deleteApiKeySchema.shape,
  toolHandler(security.delete_api_key)
);

server.tool(
  "list_audit_logs",
  "View the audit log of all actions",
  listAuditLogsSchema.shape,
  toolHandler(security.list_audit_logs)
);

// Registry tools (2, conditional)
if (registry) {
  server.tool(
    "register_in_registry",
    "Register this store in the ACE registry",
    registerInRegistrySchema.shape,
    toolHandler(registry.register_in_registry)
  );

  server.tool(
    "sync_products_to_registry",
    "Sync all published products to the registry search index",
    syncProductsToRegistrySchema.shape,
    toolHandler(registry.sync_products_to_registry)
  );
}

// Start server
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error(`ACE Seller MCP server running (store: ${config.storeId})`);
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
