#!/usr/bin/env node

import crypto from "node:crypto";
import { ShopifyAdapter } from "./adapters/shopify.js";
import { InMemoryStore } from "./server/store.js";
import { createServer, ServerConfig } from "./server/app.js";
import { SyncManager } from "./sync.js";
import { RegistryClient } from "./registry.js";

function parseArgs(args: string[]): Record<string, string> {
  const result: Record<string, string> = {};
  for (let i = 0; i < args.length; i++) {
    if (args[i].startsWith("--") && i + 1 < args.length) {
      result[args[i].slice(2)] = args[i + 1];
      i++;
    }
  }
  return result;
}

async function main() {
  const args = process.argv.slice(2);
  const platform = args[0];

  if (!platform || platform === "--help") {
    console.log(`Usage: ace-connect <platform> [options]

Platforms:
  shopify       Connect a Shopify store

Common options:
  --registry <url>       ACE Registry URL (enables discovery)
  --port <port>          Server port (default: 8081)
  --sync-interval <sec>  Sync interval in seconds (default: 300)
  --country <code>       Store country (default: US)
  --categories <list>    Comma-separated categories

Shopify options:
  --shop <domain>        Shopify store domain (required)
  --token <token>        Shopify Admin API access token (required)
  --currency <code>      Override currency (auto-detected if omitted)`);
    process.exit(0);
  }

  const opts = parseArgs(args.slice(1));

  if (platform !== "shopify") {
    console.error(`Unknown platform: ${platform}. Currently supported: shopify`);
    process.exit(1);
  }

  if (!opts.shop || !opts.token) {
    console.error("Error: --shop and --token are required for Shopify");
    process.exit(1);
  }

  const port = parseInt(opts.port || "8081");
  const syncInterval = parseInt(opts["sync-interval"] || "300");
  const country = opts.country || "US";
  const categories = opts.categories ? opts.categories.split(",") : [];
  const storeId = `ace_${crypto.randomBytes(6).toString("hex")}`;

  // Create adapter
  console.log(`[ace-connect] Connecting to Shopify: ${opts.shop}`);
  const adapter = new ShopifyAdapter(opts.shop, opts.token, opts.currency);
  const currency = await adapter.detectCurrency();
  console.log(`[ace-connect] Currency: ${currency}`);

  // Create store
  const store = new InMemoryStore();
  store.setCurrency(currency);

  // Server config
  const baseUrl = opts["base-url"] || `http://localhost:${port}`;
  const serverConfig: ServerConfig = {
    port,
    storeName: adapter.storeName,
    storeId,
    baseUrl,
    currency,
  };

  // Registry setup (before sync manager, so the callback can reference it)
  let registryClient: RegistryClient | null = null;

  if (opts.registry) {
    registryClient = new RegistryClient(
      opts.registry,
      `${baseUrl}/.well-known/agent-commerce`,
      opts.shop
    );
    await registryClient.register(categories, country);
  }

  // Create sync manager with registry callback if applicable
  const syncManager = new SyncManager(adapter, store, syncInterval, async (count) => {
    console.log(`[sync] Updated: ${count} products`);
    if (registryClient) {
      const prods = Array.from(store.products.values());
      await registryClient.syncProducts(prods, country).catch((e: any) =>
        console.error(`[registry] Sync error: ${e.message}`)
      );
    }
  });

  // Initial sync
  console.log("[ace-connect] Fetching products from Shopify...");
  const count = await syncManager.syncOnce();
  console.log(`[ace-connect] Loaded ${count} products`);

  // Push initial products to registry
  if (registryClient) {
    const products = Array.from(store.products.values());
    await registryClient.syncProducts(products, country);
  }

  // Start server
  const server = createServer(store, serverConfig);
  server.listen(port, () => {
    console.log(`[ace-connect] ACE server running on :${port}`);
    console.log(`[ace-connect] Discovery: ${baseUrl}/.well-known/agent-commerce`);
    console.log(`[ace-connect] Payment auth: ENABLED (mock)`);
  });

  // Start periodic sync
  syncManager.start();
  console.log(`[ace-connect] Sync every ${syncInterval}s`);

  // Graceful shutdown
  const shutdown = () => {
    console.log("\n[ace-connect] Shutting down...");
    syncManager.stop();
    server.close(() => {
      console.log("[ace-connect] Stopped");
      process.exit(0);
    });
  };
  process.on("SIGINT", shutdown);
  process.on("SIGTERM", shutdown);
}

main().catch((err) => {
  console.error(`Fatal: ${err.message}`);
  process.exit(1);
});
