# ACE Connect Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a CLI tool that connects a Shopify store to the ACE ecosystem with a single command — embeds an ACE-compatible HTTP server, syncs the catalog, and registers in the registry.

**Architecture:** A standalone TypeScript CLI (`ace-connect/`) with an adapter pattern. The Shopify adapter fetches products via REST API. An embedded HTTP server serves ACE endpoints. A sync loop keeps the catalog fresh. Registry integration pushes products to Elasticsearch for cross-store search.

**Tech Stack:** TypeScript, Node.js 18+ stdlib (`node:http`, `node:crypto`), native `fetch`, zero runtime dependencies.

**Spec:** `docs/superpowers/specs/2026-03-27-ace-connect-design.md`

---

## File Structure

```
ace-connect/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts              # CLI entry point, arg parsing, orchestration
│   ├── adapters/
│   │   ├── adapter.ts        # EcommerceAdapter interface + types
│   │   └── shopify.ts        # Shopify adapter
│   ├── server/
│   │   ├── app.ts            # HTTP server with ACE endpoints
│   │   ├── store.ts          # In-memory store (products, carts, orders)
│   │   └── auth.ts           # Dual auth middleware (mock payment)
│   ├── sync.ts               # Periodic sync using adapter
│   └── registry.ts           # Registry registration + product push
└── README.md
```

---

### Task 1: Project Scaffolding

**Files:**
- Create: `ace-connect/package.json`
- Create: `ace-connect/tsconfig.json`

- [ ] **Step 1: Create directories**

Run: `mkdir -p "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect/src/adapters" "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect/src/server"`

- [ ] **Step 2: Create package.json**

```json
{
  "name": "ace-connect",
  "version": "0.1.0",
  "description": "Connect any e-commerce platform to the ACE Protocol with one command",
  "type": "module",
  "main": "dist/index.js",
  "bin": {
    "ace-connect": "dist/index.js"
  },
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js"
  },
  "devDependencies": {
    "typescript": "^5.8.3",
    "@types/node": "^22.0.0"
  },
  "engines": {
    "node": ">=18.0.0"
  },
  "license": "Apache-2.0"
}
```

- [ ] **Step 3: Create tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "Node16",
    "moduleResolution": "Node16",
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "declaration": true
  },
  "include": ["src/**/*"]
}
```

- [ ] **Step 4: Install dependencies**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect" && npm install`

- [ ] **Step 5: Commit**

```bash
cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace"
git add ace-connect/package.json ace-connect/package-lock.json ace-connect/tsconfig.json
git commit -m "feat(ace-connect): scaffold TypeScript CLI project"
```

---

### Task 2: Adapter Interface + Types

**Files:**
- Create: `ace-connect/src/adapters/adapter.ts`

- [ ] **Step 1: Create adapter.ts**

```typescript
export interface AceProduct {
  id: string;
  name: string;
  description: string;
  category: string;
  tags: string[];
  price: { amount: number; currency: string };
  variants: AceVariant[];
  imageUrl: string;
  status: "published" | "draft";
  pricingModel: "fixed";
  pricePerRequest: 0;
  createdAt: string;
  updatedAt: string;
}

export interface AceVariant {
  id: string;
  name: string;
  sku: string;
  price: { amount: number; currency: string };
  inventory: number;
  attributes: Record<string, string>;
}

export interface EcommerceAdapter {
  name: string;
  storeName: string;
  fetchProducts(): Promise<AceProduct[]>;
}
```

- [ ] **Step 2: Verify**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect" && npx tsc --noEmit`

- [ ] **Step 3: Commit**

```bash
git add ace-connect/src/adapters/adapter.ts
git commit -m "feat(ace-connect): add EcommerceAdapter interface and ACE types"
```

---

### Task 3: Shopify Adapter

**Files:**
- Create: `ace-connect/src/adapters/shopify.ts`

- [ ] **Step 1: Create shopify.ts**

```typescript
import { EcommerceAdapter, AceProduct, AceVariant } from "./adapter.js";

export class ShopifyAdapter implements EcommerceAdapter {
  name = "shopify";
  storeName: string;
  private shop: string;
  private token: string;
  private currency: string;
  private apiVersion = "2025-01";

  constructor(shop: string, token: string, currency?: string) {
    this.shop = shop;
    this.storeName = shop.replace(".myshopify.com", "");
    this.token = token;
    this.currency = currency || "USD";
  }

  async detectCurrency(): Promise<string> {
    try {
      const res = await fetch(
        `https://${this.shop}/admin/api/${this.apiVersion}/shop.json`,
        { headers: this.headers() }
      );
      if (res.ok) {
        const data = await res.json() as any;
        this.currency = data.shop?.currency || this.currency;
      }
    } catch {
      // Keep default currency
    }
    return this.currency;
  }

  async fetchProducts(): Promise<AceProduct[]> {
    const products: AceProduct[] = [];
    let url: string | null =
      `https://${this.shop}/admin/api/${this.apiVersion}/products.json?status=active&limit=250`;

    while (url) {
      const res = await this.fetchWithRetry(url);
      const data = await res.json() as any;

      for (const p of data.products || []) {
        products.push(this.mapProduct(p));
      }

      url = this.parseNextLink(res.headers.get("link"));
    }

    return products;
  }

  private mapProduct(p: any): AceProduct {
    const now = new Date().toISOString();
    const firstVariant = p.variants?.[0];
    const priceCents = firstVariant
      ? Math.round(parseFloat(firstVariant.price) * 100)
      : 0;

    const variants: AceVariant[] = (p.variants || []).map((v: any, i: number) => ({
      id: `shp_v_${v.id}`,
      name: v.title || `Variant ${i + 1}`,
      sku: v.sku || "",
      price: {
        amount: Math.round(parseFloat(v.price) * 100),
        currency: this.currency,
      },
      inventory: v.inventory_quantity ?? 0,
      attributes: this.mapVariantAttributes(v, p.options),
    }));

    return {
      id: `shp_${p.id}`,
      name: p.title || "Untitled",
      description: this.stripHtml(p.body_html || ""),
      category: p.product_type || "general",
      tags: p.tags ? p.tags.split(",").map((t: string) => t.trim()).filter(Boolean) : [],
      price: { amount: priceCents, currency: this.currency },
      variants,
      imageUrl: p.images?.[0]?.src || "",
      status: "published",
      pricingModel: "fixed",
      pricePerRequest: 0,
      createdAt: p.created_at || now,
      updatedAt: now,
    };
  }

  private mapVariantAttributes(v: any, options: any[]): Record<string, string> {
    const attrs: Record<string, string> = {};
    if (options) {
      options.forEach((opt: any, i: number) => {
        const val = v[`option${i + 1}`];
        if (val && val !== "Default Title") {
          attrs[opt.name] = val;
        }
      });
    }
    return attrs;
  }

  private stripHtml(html: string): string {
    return html.replace(/<[^>]*>/g, "").replace(/\s+/g, " ").trim();
  }

  private headers(): Record<string, string> {
    return {
      "X-Shopify-Access-Token": this.token,
      Accept: "application/json",
    };
  }

  private async fetchWithRetry(url: string, retries = 3): Promise<Response> {
    for (let i = 0; i < retries; i++) {
      const res = await fetch(url, { headers: this.headers() });
      if (res.status === 429) {
        const wait = Math.pow(2, i) * 1000;
        await new Promise((r) => setTimeout(r, wait));
        continue;
      }
      if (!res.ok) {
        throw new Error(`Shopify API error (${res.status}): ${res.statusText}`);
      }
      return res;
    }
    throw new Error("Shopify API rate limit exceeded after retries");
  }

  private parseNextLink(header: string | null): string | null {
    if (!header) return null;
    const match = header.match(/<([^>]+)>;\s*rel="next"/);
    return match ? match[1] : null;
  }
}
```

- [ ] **Step 2: Verify**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect" && npx tsc --noEmit`

- [ ] **Step 3: Commit**

```bash
git add ace-connect/src/adapters/shopify.ts
git commit -m "feat(ace-connect): add Shopify adapter with rate limiting and pagination"
```

---

### Task 4: In-Memory Store

**Files:**
- Create: `ace-connect/src/server/store.ts`

- [ ] **Step 1: Create store.ts**

```typescript
import crypto from "node:crypto";
import { AceProduct } from "../adapters/adapter.js";

interface CartItem {
  product_id: string;
  variant_id: string;
  quantity: number;
  price: { amount: number; currency: string };
}

interface Cart {
  id: string;
  items: CartItem[];
  total: { amount: number; currency: string };
  created_at: string;
  updated_at: string;
}

interface OrderItem {
  product_id: string;
  product_name: string;
  variant_id: string;
  quantity: number;
  price: { amount: number; currency: string };
}

interface Order {
  id: string;
  cart_id: string;
  items: OrderItem[];
  total: { amount: number; currency: string };
  status: string;
  payment: Payment | null;
  created_at: string;
  updated_at: string;
}

interface Payment {
  id: string;
  order_id: string;
  status: string;
  provider: string;
  amount: { amount: number; currency: string };
  external_id: string;
  payment_url: string;
  created_at: string;
}

function genId(prefix: string): string {
  return `${prefix}_${crypto.randomBytes(8).toString("hex")}`;
}

export class InMemoryStore {
  products = new Map<string, AceProduct>();
  private carts = new Map<string, Cart>();
  private orders = new Map<string, Order>();
  private currency = "USD";

  setCurrency(c: string) {
    this.currency = c;
  }

  replaceProducts(products: AceProduct[]) {
    const newMap = new Map<string, AceProduct>();
    for (const p of products) newMap.set(p.id, p);
    this.products = newMap;
  }

  listProducts(query?: string, category?: string, offset = 0, limit = 20) {
    let arr = Array.from(this.products.values()).filter(
      (p) => p.status === "published"
    );
    if (category) arr = arr.filter((p) => p.category.toLowerCase() === category.toLowerCase());
    if (query) {
      const q = query.toLowerCase();
      arr = arr.filter(
        (p) => p.name.toLowerCase().includes(q) || p.description.toLowerCase().includes(q)
      );
    }
    const total = arr.length;
    return { data: arr.slice(offset, offset + limit), total };
  }

  getProduct(id: string): AceProduct | undefined {
    const p = this.products.get(id);
    return p && p.status === "published" ? p : undefined;
  }

  createCart(): Cart {
    const cart: Cart = {
      id: genId("cart"),
      items: [],
      total: { amount: 0, currency: this.currency },
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    this.carts.set(cart.id, cart);
    return cart;
  }

  getCart(id: string): Cart | undefined {
    return this.carts.get(id);
  }

  addCartItem(cartId: string, productId: string, variantId: string | undefined, quantity: number): Cart | { error: string; code: string } {
    const cart = this.carts.get(cartId);
    if (!cart) return { error: "Cart not found", code: "cart_not_found" };

    const product = this.getProduct(productId);
    if (!product) return { error: "Product not found", code: "product_not_found" };

    let price = product.price;
    if (variantId) {
      const variant = product.variants.find((v) => v.id === variantId);
      if (!variant) return { error: "Variant not found", code: "variant_not_found" };
      price = variant.price;
    } else if (product.variants.length > 0) {
      price = product.variants[0].price;
    }

    cart.items.push({ product_id: productId, variant_id: variantId || "", quantity, price });
    cart.total.amount = cart.items.reduce((sum, i) => sum + i.price.amount * i.quantity, 0);
    cart.updated_at = new Date().toISOString();
    return cart;
  }

  createOrder(cartId: string): Order | { error: string; code: string } {
    const cart = this.carts.get(cartId);
    if (!cart) return { error: "Cart not found", code: "cart_not_found" };
    if (cart.items.length === 0) return { error: "Cart is empty", code: "empty_cart" };

    const items: OrderItem[] = cart.items.map((i) => ({
      product_id: i.product_id,
      product_name: this.products.get(i.product_id)?.name || "Unknown",
      variant_id: i.variant_id,
      quantity: i.quantity,
      price: i.price,
    }));

    const order: Order = {
      id: genId("ord"),
      cart_id: cartId,
      items,
      total: { ...cart.total },
      status: "pending",
      payment: null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    this.orders.set(order.id, order);
    return order;
  }

  getOrder(id: string): Order | undefined {
    return this.orders.get(id);
  }

  payOrder(orderId: string, provider: string): Payment | { error: string; code: string } {
    const order = this.orders.get(orderId);
    if (!order) return { error: "Order not found", code: "order_not_found" };
    if (order.status !== "pending") return { error: "Order not pending", code: "invalid_status" };

    const payment: Payment = {
      id: genId("pay"),
      order_id: orderId,
      status: "completed",
      provider,
      amount: order.total,
      external_id: `mock_${orderId}`,
      payment_url: `https://pay.example.com/mock/${orderId}`,
      created_at: new Date().toISOString(),
    };

    order.payment = payment;
    order.status = "paid";
    order.updated_at = new Date().toISOString();
    return payment;
  }

  getPaymentByOrderId(orderId: string): Payment | undefined {
    return this.orders.get(orderId)?.payment || undefined;
  }
}
```

- [ ] **Step 2: Verify**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect" && npx tsc --noEmit`

- [ ] **Step 3: Commit**

```bash
git add ace-connect/src/server/store.ts
git commit -m "feat(ace-connect): add in-memory store for products, carts, orders"
```

---

### Task 5: Auth Middleware

**Files:**
- Create: `ace-connect/src/server/auth.ts`

- [ ] **Step 1: Create auth.ts**

```typescript
import http from "node:http";

export function authenticate(req: http.IncomingMessage): { ok: boolean; actor: string; status?: number; body?: any } {
  // Accept X-ACE-Payment: mock:*
  const payment = req.headers["x-ace-payment"] as string | undefined;
  if (payment) {
    const idx = payment.indexOf(":");
    if (idx > 0) {
      const provider = payment.slice(0, idx);
      if (provider === "mock") {
        return { ok: true, actor: `payment:${payment.slice(idx + 1)}` };
      }
    }
    return { ok: false, actor: "", status: 401, body: { error: "Payment provider not supported", code: "payment_rejected" } };
  }

  // Accept X-ACE-Key (any non-empty value)
  const key = req.headers["x-ace-key"] as string | undefined;
  if (key) {
    return { ok: true, actor: `key:${key.slice(0, 8)}` };
  }

  // No auth → 402
  return {
    ok: false,
    actor: "",
    status: 402,
    body: {
      error: "Payment or API key required",
      code: "payment_required",
      pricing: {
        price: 0,
        currency: "USD",
        accepted_providers: ["mock"],
      },
    },
  };
}
```

- [ ] **Step 2: Verify**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect" && npx tsc --noEmit`

- [ ] **Step 3: Commit**

```bash
git add ace-connect/src/server/auth.ts
git commit -m "feat(ace-connect): add auth middleware (mock payment + API key)"
```

---

### Task 6: HTTP Server with ACE Endpoints

**Files:**
- Create: `ace-connect/src/server/app.ts`

- [ ] **Step 1: Create app.ts**

This is the largest file. It creates an HTTP server with all ACE endpoints. Key patterns:

- Uses `node:http` directly, routes via `req.method` + `req.url` matching
- Public endpoints: `/.well-known/agent-commerce`, `/ace/v1/pricing`
- Auth-protected endpoints: everything else under `/ace/v1/`
- JSON helpers: `writeJSON(res, status, body)`, `readBody(req)`
- Pricing headers on all responses: `X-ACE-Price: 0.00`, `X-ACE-Currency: USD`

```typescript
import http from "node:http";
import { URL } from "node:url";
import { InMemoryStore } from "./store.js";
import { authenticate } from "./auth.js";

export interface ServerConfig {
  port: number;
  storeName: string;
  storeId: string;
  baseUrl: string;
  currency: string;
}

function writeJSON(res: http.ServerResponse, status: number, body: any) {
  res.setHeader("Content-Type", "application/json");
  res.setHeader("X-ACE-Price", "0.00");
  res.setHeader("X-ACE-Currency", "USD");
  res.writeHead(status);
  res.end(JSON.stringify(body));
}

function writeError(res: http.ServerResponse, status: number, code: string, message: string) {
  res.setHeader("Content-Type", "application/json");
  res.writeHead(status);
  res.end(JSON.stringify({ error: message, code }));
}

function readBody(req: http.IncomingMessage): Promise<any> {
  return new Promise((resolve, reject) => {
    let data = "";
    req.on("data", (chunk) => (data += chunk));
    req.on("end", () => {
      try { resolve(data ? JSON.parse(data) : {}); }
      catch { reject(new Error("Invalid JSON")); }
    });
    req.on("error", reject);
  });
}

export function createServer(store: InMemoryStore, config: ServerConfig): http.Server {
  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url || "/", `http://localhost:${config.port}`);
    const path = url.pathname;
    const method = req.method || "GET";

    try {
      // Public endpoints
      if (method === "GET" && path === "/.well-known/agent-commerce") {
        return writeJSON(res, 200, {
          store_id: config.storeId,
          name: config.storeName,
          version: "1.0.0",
          ace_base_url: `${config.baseUrl}/ace/v1`,
          capabilities: ["catalog", "cart", "orders", "payments", "shipping"],
          auth: { type: "api_key", header: "X-ACE-Key" },
          payment_auth: {
            enabled: true,
            header: "X-ACE-Payment",
            providers: ["mock"],
            default_currency: config.currency,
          },
          currencies: [config.currency],
        });
      }

      if (method === "GET" && path === "/ace/v1/pricing") {
        return writeJSON(res, 200, {
          default_currency: config.currency,
          endpoints: [
            { method: "GET", path: "/ace/v1/products", price: 0 },
            { method: "GET", path: "/ace/v1/products/{id}", price: 0 },
            { method: "POST", path: "/ace/v1/cart", price: 0 },
            { method: "POST", path: "/ace/v1/cart/{id}/items", price: 0 },
            { method: "POST", path: "/ace/v1/orders", price: 0 },
            { method: "POST", path: "/ace/v1/orders/{id}/pay", price: 0 },
          ],
        });
      }

      // Auth-protected endpoints
      if (path.startsWith("/ace/v1/")) {
        const auth = authenticate(req);
        if (!auth.ok) {
          return writeJSON(res, auth.status!, auth.body);
        }
      }

      // Routes
      if (method === "GET" && path === "/ace/v1/products") {
        const q = url.searchParams.get("q") || undefined;
        const cat = url.searchParams.get("category") || undefined;
        const offset = parseInt(url.searchParams.get("offset") || "0");
        const limit = Math.min(parseInt(url.searchParams.get("limit") || "20"), 100);
        const result = store.listProducts(q, cat, offset, limit);
        return writeJSON(res, 200, { ...result, offset, limit });
      }

      const productMatch = path.match(/^\/ace\/v1\/products\/(.+)$/);
      if (method === "GET" && productMatch) {
        const product = store.getProduct(productMatch[1]);
        if (!product) return writeError(res, 404, "not_found", "Product not found");
        return writeJSON(res, 200, product);
      }

      if (method === "POST" && path === "/ace/v1/cart") {
        return writeJSON(res, 201, store.createCart());
      }

      const cartItemMatch = path.match(/^\/ace\/v1\/cart\/(.+)\/items$/);
      if (method === "POST" && cartItemMatch) {
        const body = await readBody(req);
        const result = store.addCartItem(cartItemMatch[1], body.product_id, body.variant_id, body.quantity || 1);
        if ("error" in result) return writeError(res, 400, result.code, result.error);
        return writeJSON(res, 200, result);
      }

      const cartMatch = path.match(/^\/ace\/v1\/cart\/(.+)$/);
      if (method === "GET" && cartMatch && !cartMatch[1].includes("/")) {
        const cart = store.getCart(cartMatch[1]);
        if (!cart) return writeError(res, 404, "not_found", "Cart not found");
        return writeJSON(res, 200, cart);
      }

      if (method === "POST" && path === "/ace/v1/orders") {
        const body = await readBody(req);
        const result = store.createOrder(body.cart_id);
        if ("error" in result) return writeError(res, 400, result.code, result.error);
        return writeJSON(res, 201, result);
      }

      const payMatch = path.match(/^\/ace\/v1\/orders\/(.+)\/pay$/);
      if (method === "POST" && payMatch) {
        const body = await readBody(req);
        const result = store.payOrder(payMatch[1], body.provider || "mock");
        if ("error" in result) return writeError(res, 400, result.code, result.error);
        return writeJSON(res, 201, result);
      }

      const payStatusMatch = path.match(/^\/ace\/v1\/orders\/(.+)\/pay\/status$/);
      if (method === "GET" && payStatusMatch) {
        const payment = store.getPaymentByOrderId(payStatusMatch[1]);
        if (!payment) return writeError(res, 404, "not_found", "Payment not found");
        return writeJSON(res, 200, payment);
      }

      const orderMatch = path.match(/^\/ace\/v1\/orders\/(.+)$/);
      if (method === "GET" && orderMatch && !orderMatch[1].includes("/")) {
        const order = store.getOrder(orderMatch[1]);
        if (!order) return writeError(res, 404, "not_found", "Order not found");
        return writeJSON(res, 200, order);
      }

      if (method === "POST" && path === "/ace/v1/shipping/quote") {
        return writeJSON(res, 200, {
          options: [
            { id: "ship_standard", name: "Standard Shipping", price: { amount: 599, currency: config.currency }, estimated_days: 7 },
            { id: "ship_express", name: "Express Shipping", price: { amount: 1299, currency: config.currency }, estimated_days: 3 },
            { id: "ship_overnight", name: "Overnight Shipping", price: { amount: 2499, currency: config.currency }, estimated_days: 1 },
          ],
        });
      }

      writeError(res, 404, "not_found", "Endpoint not found");
    } catch (err: any) {
      writeError(res, 500, "internal_error", err.message || "Internal server error");
    }
  });

  return server;
}
```

- [ ] **Step 2: Verify**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect" && npx tsc --noEmit`

- [ ] **Step 3: Commit**

```bash
git add ace-connect/src/server/app.ts
git commit -m "feat(ace-connect): add embedded ACE HTTP server with all endpoints"
```

---

### Task 7: Sync + Registry

**Files:**
- Create: `ace-connect/src/sync.ts`
- Create: `ace-connect/src/registry.ts`

- [ ] **Step 1: Create sync.ts**

```typescript
import { EcommerceAdapter } from "./adapters/adapter.js";
import { InMemoryStore } from "./server/store.js";

export class SyncManager {
  private adapter: EcommerceAdapter;
  private store: InMemoryStore;
  private intervalMs: number;
  private timer: NodeJS.Timeout | null = null;
  private onSync?: (count: number) => void;

  constructor(adapter: EcommerceAdapter, store: InMemoryStore, intervalSec: number, onSync?: (count: number) => void) {
    this.adapter = adapter;
    this.store = store;
    this.intervalMs = intervalSec * 1000;
    this.onSync = onSync;
  }

  async syncOnce(): Promise<number> {
    const products = await this.adapter.fetchProducts();
    this.store.replaceProducts(products);
    this.onSync?.(products.length);
    return products.length;
  }

  start() {
    this.timer = setInterval(() => {
      this.syncOnce().catch((err) =>
        console.error(`[sync] Error: ${err.message}`)
      );
    }, this.intervalMs);
  }

  stop() {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }
}
```

- [ ] **Step 2: Create registry.ts**

```typescript
import fs from "node:fs";
import path from "node:path";
import { AceProduct } from "./adapters/adapter.js";

interface RegistryState {
  shop: string;
  store_id: string;
  registry_token: string;
  registered_at: string;
}

const STATE_FILE = ".ace-connect.json";

export class RegistryClient {
  private registryUrl: string;
  private wellKnownUrl: string;
  private shop: string;
  private storeId: string | null = null;
  private registryToken: string | null = null;

  constructor(registryUrl: string, wellKnownUrl: string, shop: string) {
    this.registryUrl = registryUrl;
    this.wellKnownUrl = wellKnownUrl;
    this.shop = shop;
  }

  async register(categories: string[], country: string): Promise<void> {
    // Try loading persisted state
    const saved = this.loadState();
    if (saved && saved.shop === this.shop) {
      this.storeId = saved.store_id;
      this.registryToken = saved.registry_token;
      console.log(`[registry] Reusing registration: ${this.storeId}`);
      return;
    }

    const res = await fetch(`${this.registryUrl}/registry/v1/stores`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        well_known_url: this.wellKnownUrl,
        categories,
        country,
      }),
    });

    if (!res.ok) {
      const body = await res.text();
      throw new Error(`Registry registration failed (${res.status}): ${body}`);
    }

    const data = await res.json() as any;
    this.storeId = data.id;
    this.registryToken = data.registry_token;

    this.saveState({
      shop: this.shop,
      store_id: this.storeId!,
      registry_token: this.registryToken!,
      registered_at: new Date().toISOString(),
    });

    console.log(`[registry] Registered as ${this.storeId}`);
  }

  async syncProducts(products: AceProduct[], country: string): Promise<void> {
    if (!this.registryToken) return;

    const payload = {
      products: products.map((p) => ({
        product_id: p.id,
        name: p.name,
        description: p.description,
        category: p.category,
        tags: p.tags,
        price_range: {
          min: Math.min(p.price.amount, ...p.variants.map((v) => v.price.amount)),
          max: Math.max(p.price.amount, ...p.variants.map((v) => v.price.amount)),
          currency: p.price.currency,
        },
        variants_summary: p.variants.map((v) => v.name),
        image_url: p.imageUrl,
        in_stock: p.variants.length === 0 || p.variants.some((v) => v.inventory > 0),
        rating: { average: 0, count: 0 },
        location: { country, region: "" },
      })),
    };

    const res = await fetch(`${this.registryUrl}/registry/v1/products/sync`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.registryToken}`,
      },
      body: JSON.stringify(payload),
    });

    if (res.ok) {
      const data = await res.json() as any;
      console.log(`[registry] Synced ${data.indexed || 0} products`);
    } else {
      console.error(`[registry] Sync failed (${res.status})`);
    }
  }

  private loadState(): RegistryState | null {
    try {
      const data = fs.readFileSync(STATE_FILE, "utf-8");
      return JSON.parse(data);
    } catch {
      return null;
    }
  }

  private saveState(state: RegistryState) {
    fs.writeFileSync(STATE_FILE, JSON.stringify(state, null, 2));
  }
}
```

- [ ] **Step 3: Verify**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect" && npx tsc --noEmit`

- [ ] **Step 4: Commit**

```bash
git add ace-connect/src/sync.ts ace-connect/src/registry.ts
git commit -m "feat(ace-connect): add sync manager and registry client with persistence"
```

---

### Task 8: CLI Entry Point

**Files:**
- Create: `ace-connect/src/index.ts`

- [ ] **Step 1: Create index.ts**

```typescript
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

  // Initial sync
  console.log("[ace-connect] Fetching products from Shopify...");
  const syncManager = new SyncManager(adapter, store, syncInterval, (count) => {
    console.log(`[sync] Updated: ${count} products`);
  });
  const count = await syncManager.syncOnce();
  console.log(`[ace-connect] Loaded ${count} products`);

  // Server config
  const baseUrl = opts["base-url"] || `http://localhost:${port}`;
  const serverConfig: ServerConfig = {
    port,
    storeName: adapter.storeName,
    storeId,
    baseUrl,
    currency,
  };

  // Start server
  const server = createServer(store, serverConfig);
  server.listen(port, () => {
    console.log(`[ace-connect] ACE server running on :${port}`);
    console.log(`[ace-connect] Discovery: ${baseUrl}/.well-known/agent-commerce`);
    console.log(`[ace-connect] Payment auth: ENABLED (mock)`);
  });

  // Registry
  if (opts.registry) {
    const registryClient = new RegistryClient(
      opts.registry,
      `${baseUrl}/.well-known/agent-commerce`,
      opts.shop
    );
    await registryClient.register(categories, country);

    const products = Array.from(store.products.values());
    await registryClient.syncProducts(products, country);

    // Also sync on each product refresh
    syncManager = Object.assign(syncManager, {
      onSync: async (c: number) => {
        console.log(`[sync] Updated: ${c} products`);
        const prods = Array.from(store.products.values());
        await registryClient.syncProducts(prods, country).catch((e: any) =>
          console.error(`[registry] Sync error: ${e.message}`)
        );
      },
    });
  }

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
```

- [ ] **Step 2: Build**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect" && npm run build`

- [ ] **Step 3: Commit**

```bash
git add ace-connect/src/index.ts
git commit -m "feat(ace-connect): add CLI entry point with Shopify integration"
```

---

### Task 9: README

**Files:**
- Create: `ace-connect/README.md`

- [ ] **Step 1: Create README**

Cover: what it is, quick start with Shopify, CLI options, how it works (diagram), adapter architecture for future platforms, development instructions.

- [ ] **Step 2: Commit**

```bash
git add ace-connect/README.md
git commit -m "docs(ace-connect): add README with Shopify quick start"
```

---

### Task 10: Build, Test, Push

- [ ] **Step 1: Full build**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/ace-connect" && npm run build`

- [ ] **Step 2: Test locally** (without real Shopify — just verify server starts)

Run: Temporarily test that the server starts with mock data by creating a small test or running with fake args and checking it errors correctly.

- [ ] **Step 3: Update main README**

Add ace-connect to the repo structure and Phase 3 in the roadmap.

- [ ] **Step 4: Push**

```bash
cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace"
git add -A
git commit -m "feat(ace-connect): complete Shopify connector v0.1.0"
gh auth switch --user nicoroldan1
git push origin main
gh auth switch --user nicroldan_meli
```
