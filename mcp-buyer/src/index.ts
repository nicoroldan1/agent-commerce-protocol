#!/usr/bin/env node

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { loadConfig } from "./config.js";
import { StoreClient } from "./client/store.js";
import { RegistryClient } from "./client/registry.js";
import {
  createDiscoveryTools,
  discoverStoresSchema,
  searchProductsSchema,
} from "./tools/discovery.js";
import {
  createCatalogTools,
  browseStoreSchema,
  getProductSchema,
  getPricingSchema,
} from "./tools/catalog.js";
import {
  createPurchaseTools,
  createCartSchema,
  getCartSchema,
  addToCartSchema,
  shippingQuoteSchema,
  placeOrderSchema,
  payOrderSchema,
  getOrderSchema,
  paymentStatusSchema,
} from "./tools/purchase.js";

const config = loadConfig();
const storeClient = new StoreClient(config);
const registryClient = config.registryUrl
  ? new RegistryClient(config.registryUrl)
  : null;

const discovery = createDiscoveryTools(registryClient);
const catalog = createCatalogTools(storeClient, config);
const purchase = createPurchaseTools(storeClient, config);

const server = new McpServer({
  name: "ace-buyer",
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

// Discovery tools (only if registry is configured)
if (registryClient) {
  server.tool(
    "discover_stores",
    "Search for ACE stores in the registry by name, category, or country",
    discoverStoresSchema.shape,
    toolHandler(discovery.discover_stores)
  );

  server.tool(
    "search_products",
    "Search for products across all stores in the registry",
    searchProductsSchema.shape,
    toolHandler(discovery.search_products)
  );
}

// Catalog tools
server.tool(
  "browse_store",
  "List products from an ACE store with optional filters",
  browseStoreSchema.shape,
  toolHandler(catalog.browse_store)
);

server.tool(
  "get_product",
  "Get full product details including variants and pricing",
  getProductSchema.shape,
  toolHandler(catalog.get_product)
);

server.tool(
  "get_pricing",
  "Get a store's pricing schedule for all endpoints",
  getPricingSchema.shape,
  toolHandler(catalog.get_pricing)
);

// Purchase tools
server.tool(
  "create_cart",
  "Create a new shopping cart at an ACE store",
  createCartSchema.shape,
  toolHandler(purchase.create_cart)
);

server.tool(
  "get_cart",
  "Get the current state of a shopping cart",
  getCartSchema.shape,
  toolHandler(purchase.get_cart)
);

server.tool(
  "add_to_cart",
  "Add a product to a shopping cart",
  addToCartSchema.shape,
  toolHandler(purchase.add_to_cart)
);

server.tool(
  "shipping_quote",
  "Get shipping options and prices for a destination",
  shippingQuoteSchema.shape,
  toolHandler(purchase.shipping_quote)
);

server.tool(
  "place_order",
  "Convert a cart into an order",
  placeOrderSchema.shape,
  toolHandler(purchase.place_order)
);

server.tool(
  "pay_order",
  "Pay for an order using the configured payment provider",
  payOrderSchema.shape,
  toolHandler(purchase.pay_order)
);

server.tool(
  "get_order",
  "Get full order details including items and payment status",
  getOrderSchema.shape,
  toolHandler(purchase.get_order)
);

server.tool(
  "payment_status",
  "Check payment status for an order",
  paymentStatusSchema.shape,
  toolHandler(purchase.payment_status)
);

// Start server
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error(`ACE Buyer MCP server running (${registryClient ? "registry + store" : "store only"} mode)`);
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
