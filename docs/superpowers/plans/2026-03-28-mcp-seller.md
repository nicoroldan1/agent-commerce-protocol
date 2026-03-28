# ACE Seller MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a TypeScript MCP server that lets store owners manage their ACE store via Claude — catalog, orders, policies, API keys, audit logs, bulk import, and registry sync.

**Architecture:** Standalone TypeScript project (`mcp-seller/`) mirroring the mcp-buyer structure. A single HTTP client with admin Bearer auth talks to the ACE store Admin API. Tools organized by domain: catalog, orders, policies, security, registry.

**Tech Stack:** TypeScript, `@modelcontextprotocol/sdk`, `zod`, native `fetch`.

**Spec:** `docs/superpowers/specs/2026-03-27-mcp-seller-design.md`

---

## File Structure

```
mcp-seller/
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts              # MCP server entry point, tool registration
│   ├── config.ts             # Env var parsing
│   ├── client.ts             # HTTP client with admin Bearer auth
│   └── tools/
│       ├── catalog.ts        # 8 tools
│       ├── orders.ts         # 4 tools
│       ├── policies.ts       # 5 tools
│       ├── security.ts       # 4 tools
│       └── registry.ts       # 2 tools
└── README.md
```

---

### Task 1: Project Scaffolding

**Files:**
- Create: `mcp-seller/package.json`
- Create: `mcp-seller/tsconfig.json`

- [ ] **Step 1: Create directories and files**

```bash
mkdir -p "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/mcp-seller/src/tools"
```

Create `mcp-seller/package.json`:
```json
{
  "name": "ace-seller-mcp",
  "version": "0.1.0",
  "description": "MCP server for ACE Protocol store owners — manage catalog, orders, policies, and API keys",
  "type": "module",
  "main": "dist/index.js",
  "bin": { "ace-seller-mcp": "dist/index.js" },
  "scripts": { "build": "tsc", "start": "node dist/index.js" },
  "dependencies": {
    "@modelcontextprotocol/sdk": "^1.12.1",
    "zod": "^3.24.4"
  },
  "devDependencies": {
    "typescript": "^5.8.3",
    "@types/node": "^22.0.0"
  },
  "engines": { "node": ">=18.0.0" },
  "license": "Apache-2.0"
}
```

Create `mcp-seller/tsconfig.json`:
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

- [ ] **Step 2: Install deps**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/mcp-seller" && npm install`

- [ ] **Step 3: Commit**

```bash
git add mcp-seller/package.json mcp-seller/package-lock.json mcp-seller/tsconfig.json
git commit -m "feat(mcp-seller): scaffold TypeScript MCP project"
```

---

### Task 2: Config + HTTP Client

**Files:**
- Create: `mcp-seller/src/config.ts`
- Create: `mcp-seller/src/client.ts`

- [ ] **Step 1: Create config.ts**

```typescript
export interface Config {
  storeUrl: string;
  adminToken: string;
  storeId: string;
  registryUrl: string | null;
}

export function loadConfig(): Config {
  const storeUrl = process.env.ACE_STORE_URL;
  const adminToken = process.env.ACE_ADMIN_TOKEN;
  const storeId = process.env.ACE_STORE_ID;

  if (!storeUrl || !adminToken || !storeId) {
    throw new Error("ACE_STORE_URL, ACE_ADMIN_TOKEN, and ACE_STORE_ID are all required.");
  }

  return {
    storeUrl,
    adminToken,
    storeId,
    registryUrl: process.env.ACE_REGISTRY_URL || null,
  };
}
```

- [ ] **Step 2: Create client.ts**

```typescript
import { Config } from "./config.js";

export class AdminClient {
  private config: Config;

  constructor(config: Config) {
    this.config = config;
  }

  private url(path: string): string {
    return `${this.config.storeUrl}${path.replace("{store_id}", this.config.storeId)}`;
  }

  private headers(): Record<string, string> {
    return {
      Authorization: `Bearer ${this.config.adminToken}`,
      "Content-Type": "application/json",
      Accept: "application/json",
    };
  }

  async get(path: string, params?: Record<string, string>): Promise<any> {
    const u = new URL(this.url(path));
    if (params) {
      Object.entries(params).forEach(([k, v]) => {
        if (v !== undefined && v !== "") u.searchParams.set(k, v);
      });
    }
    const res = await fetch(u.toString(), { headers: this.headers() });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(`Admin API error (${res.status}): ${body.error || res.statusText} [code: ${body.code || "unknown"}]`);
    }
    return res.json();
  }

  async post(path: string, body?: any): Promise<any> {
    const res = await fetch(this.url(path), {
      method: "POST",
      headers: this.headers(),
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
      const respBody = await res.json().catch(() => ({}));
      throw new Error(`Admin API error (${res.status}): ${respBody.error || res.statusText} [code: ${respBody.code || "unknown"}]`);
    }
    return res.json();
  }

  async patch(path: string, body: any): Promise<any> {
    const res = await fetch(this.url(path), {
      method: "PATCH",
      headers: this.headers(),
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const respBody = await res.json().catch(() => ({}));
      throw new Error(`Admin API error (${res.status}): ${respBody.error || res.statusText} [code: ${respBody.code || "unknown"}]`);
    }
    return res.json();
  }

  async put(path: string, body: any): Promise<any> {
    const res = await fetch(this.url(path), {
      method: "PUT",
      headers: this.headers(),
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const respBody = await res.json().catch(() => ({}));
      throw new Error(`Admin API error (${res.status}): ${respBody.error || res.statusText} [code: ${respBody.code || "unknown"}]`);
    }
    return res.json();
  }

  async delete(path: string): Promise<void> {
    const res = await fetch(this.url(path), {
      method: "DELETE",
      headers: this.headers(),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(`Admin API error (${res.status}): ${body.error || res.statusText} [code: ${body.code || "unknown"}]`);
    }
  }
}
```

- [ ] **Step 3: Verify**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/mcp-seller" && npx tsc --noEmit`

- [ ] **Step 4: Commit**

```bash
git add mcp-seller/src/config.ts mcp-seller/src/client.ts
git commit -m "feat(mcp-seller): add config and admin HTTP client"
```

---

### Task 3: Catalog Tools (8 tools)

**Files:**
- Create: `mcp-seller/src/tools/catalog.ts`

- [ ] **Step 1: Create catalog.ts**

```typescript
import { z } from "zod";
import { AdminClient } from "../client.js";
import { Config } from "../config.js";

const variantSchema = z.object({
  name: z.string(),
  sku: z.string().optional(),
  price: z.object({ amount: z.number(), currency: z.string() }),
  inventory: z.number(),
  attributes: z.record(z.string()).optional(),
});

export const listProductsSchema = z.object({
  offset: z.number().optional(),
  limit: z.number().optional(),
});

export const createProductSchema = z.object({
  name: z.string(),
  description: z.string(),
  price: z.object({ amount: z.number(), currency: z.string() }),
  variants: z.array(variantSchema).optional(),
});

export const bulkCreateProductsSchema = z.object({
  products: z.array(z.object({
    name: z.string(),
    description: z.string(),
    price: z.object({ amount: z.number(), currency: z.string() }),
    variants: z.array(variantSchema).optional(),
  })),
});

export const updateProductSchema = z.object({
  product_id: z.string(),
  name: z.string().optional(),
  description: z.string().optional(),
  price: z.object({ amount: z.number(), currency: z.string() }).optional(),
});

export const deleteProductSchema = z.object({ product_id: z.string() });
export const publishProductSchema = z.object({ product_id: z.string() });
export const unpublishProductSchema = z.object({ product_id: z.string() });

export const updateInventorySchema = z.object({
  variant_id: z.string(),
  inventory: z.number(),
});

export function createCatalogTools(client: AdminClient, config: Config) {
  const base = `/api/v1/stores/{store_id}`;

  return {
    async list_products(input: z.infer<typeof listProductsSchema>) {
      const params: Record<string, string> = {};
      if (input.offset !== undefined) params.offset = String(input.offset);
      if (input.limit !== undefined) params.limit = String(input.limit);
      return client.get(`${base}/products`, params);
    },

    async create_product(input: z.infer<typeof createProductSchema>) {
      return client.post(`${base}/products`, input);
    },

    async bulk_create_products(input: z.infer<typeof bulkCreateProductsSchema>) {
      const result = { created: 0, errors: [] as Array<{ index: number; name: string; error: string }> };
      for (let i = 0; i < input.products.length; i++) {
        try {
          await client.post(`${base}/products`, input.products[i]);
          result.created++;
        } catch (err: any) {
          result.errors.push({ index: i, name: input.products[i].name, error: err.message });
        }
      }
      return result;
    },

    async update_product(input: z.infer<typeof updateProductSchema>) {
      const { product_id, ...updates } = input;
      return client.patch(`${base}/products/${product_id}`, updates);
    },

    async delete_product(input: z.infer<typeof deleteProductSchema>) {
      await client.delete(`${base}/products/${input.product_id}`);
      return { deleted: true };
    },

    async publish_product(input: z.infer<typeof publishProductSchema>) {
      return client.post(`${base}/products/${input.product_id}/publish`);
    },

    async unpublish_product(input: z.infer<typeof unpublishProductSchema>) {
      return client.post(`${base}/products/${input.product_id}/unpublish`);
    },

    async update_inventory(input: z.infer<typeof updateInventorySchema>) {
      await client.patch(`${base}/variants/${input.variant_id}/inventory`, { inventory: input.inventory });
      return { updated: true };
    },
  };
}
```

- [ ] **Step 2: Verify and commit**

```bash
npx tsc --noEmit
git add mcp-seller/src/tools/catalog.ts
git commit -m "feat(mcp-seller): add 8 catalog tools (CRUD, bulk, publish, inventory)"
```

---

### Task 4: Orders Tools (4 tools)

**Files:**
- Create: `mcp-seller/src/tools/orders.ts`

- [ ] **Step 1: Create orders.ts**

```typescript
import { z } from "zod";
import { AdminClient } from "../client.js";
import { Config } from "../config.js";

export const listOrdersSchema = z.object({
  offset: z.number().optional(),
  limit: z.number().optional(),
});
export const getOrderSchema = z.object({ order_id: z.string() });
export const fulfillOrderSchema = z.object({ order_id: z.string() });
export const refundOrderSchema = z.object({ order_id: z.string() });

export function createOrderTools(client: AdminClient, config: Config) {
  const base = `/api/v1/stores/{store_id}`;

  return {
    async list_orders(input: z.infer<typeof listOrdersSchema>) {
      const params: Record<string, string> = {};
      if (input.offset !== undefined) params.offset = String(input.offset);
      if (input.limit !== undefined) params.limit = String(input.limit);
      return client.get(`${base}/orders`, params);
    },

    async get_order(input: z.infer<typeof getOrderSchema>) {
      return client.get(`${base}/orders/${input.order_id}`);
    },

    async fulfill_order(input: z.infer<typeof fulfillOrderSchema>) {
      return client.post(`${base}/orders/${input.order_id}/fulfill`);
    },

    async refund_order(input: z.infer<typeof refundOrderSchema>) {
      return client.post(`${base}/orders/${input.order_id}/refund`);
    },
  };
}
```

- [ ] **Step 2: Verify and commit**

```bash
npx tsc --noEmit
git add mcp-seller/src/tools/orders.ts
git commit -m "feat(mcp-seller): add 4 order tools (list, get, fulfill, refund)"
```

---

### Task 5: Policies Tools (5 tools)

**Files:**
- Create: `mcp-seller/src/tools/policies.ts`

- [ ] **Step 1: Create policies.ts**

```typescript
import { z } from "zod";
import { AdminClient } from "../client.js";
import { Config } from "../config.js";

export const getPoliciesSchema = z.object({});
export const updatePoliciesSchema = z.object({
  policies: z.array(z.object({
    action: z.string(),
    effect: z.enum(["allow", "deny", "approval"]),
  })),
});
export const listApprovalsSchema = z.object({});
export const approveActionSchema = z.object({ approval_id: z.string() });
export const rejectActionSchema = z.object({ approval_id: z.string() });

export function createPolicyTools(client: AdminClient, config: Config) {
  const base = `/api/v1/stores/{store_id}`;

  return {
    async get_policies(_input: z.infer<typeof getPoliciesSchema>) {
      return client.get(`${base}/policies`);
    },

    async update_policies(input: z.infer<typeof updatePoliciesSchema>) {
      return client.put(`${base}/policies`, input.policies);
    },

    async list_approvals(_input: z.infer<typeof listApprovalsSchema>) {
      return client.get(`${base}/approvals`);
    },

    async approve_action(input: z.infer<typeof approveActionSchema>) {
      return client.post(`${base}/approvals/${input.approval_id}/approve`);
    },

    async reject_action(input: z.infer<typeof rejectActionSchema>) {
      return client.post(`${base}/approvals/${input.approval_id}/reject`);
    },
  };
}
```

- [ ] **Step 2: Verify and commit**

```bash
npx tsc --noEmit
git add mcp-seller/src/tools/policies.ts
git commit -m "feat(mcp-seller): add 5 policy tools (policies, approvals)"
```

---

### Task 6: Security Tools (4 tools)

**Files:**
- Create: `mcp-seller/src/tools/security.ts`

- [ ] **Step 1: Create security.ts**

```typescript
import { z } from "zod";
import { AdminClient } from "../client.js";
import { Config } from "../config.js";

export const listApiKeysSchema = z.object({});
export const createApiKeySchema = z.object({
  name: z.string(),
  scopes: z.array(z.string()),
});
export const deleteApiKeySchema = z.object({ key_id: z.string() });
export const listAuditLogsSchema = z.object({
  action: z.string().optional(),
  actor: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

export function createSecurityTools(client: AdminClient, config: Config) {
  const base = `/api/v1/stores/{store_id}`;

  return {
    async list_api_keys(_input: z.infer<typeof listApiKeysSchema>) {
      return client.get(`${base}/api-keys`);
    },

    async create_api_key(input: z.infer<typeof createApiKeySchema>) {
      return client.post(`${base}/api-keys`, input);
    },

    async delete_api_key(input: z.infer<typeof deleteApiKeySchema>) {
      await client.delete(`${base}/api-keys/${input.key_id}`);
      return { deleted: true };
    },

    async list_audit_logs(input: z.infer<typeof listAuditLogsSchema>) {
      const params: Record<string, string> = {};
      if (input.action) params.action = input.action;
      if (input.actor) params.actor = input.actor;
      if (input.offset !== undefined) params.offset = String(input.offset);
      if (input.limit !== undefined) params.limit = String(input.limit);
      return client.get(`${base}/audit-logs`, params);
    },
  };
}
```

- [ ] **Step 2: Verify and commit**

```bash
npx tsc --noEmit
git add mcp-seller/src/tools/security.ts
git commit -m "feat(mcp-seller): add 4 security tools (api-keys, audit-logs)"
```

---

### Task 7: Registry Tools (2 tools)

**Files:**
- Create: `mcp-seller/src/tools/registry.ts`

- [ ] **Step 1: Create registry.ts**

```typescript
import { z } from "zod";
import fs from "node:fs";
import { AdminClient } from "../client.js";
import { Config } from "../config.js";

const STATE_FILE = ".ace-seller.json";

export const registerInRegistrySchema = z.object({
  categories: z.array(z.string()).optional(),
  country: z.string().optional(),
});

export const syncProductsToRegistrySchema = z.object({});

interface RegistryState {
  store_url: string;
  store_id: string;
  registry_token: string;
  registered_at: string;
}

function loadState(): RegistryState | null {
  try {
    return JSON.parse(fs.readFileSync(STATE_FILE, "utf-8"));
  } catch {
    return null;
  }
}

function saveState(state: RegistryState) {
  fs.writeFileSync(STATE_FILE, JSON.stringify(state, null, 2));
}

export function createRegistryTools(client: AdminClient, config: Config) {
  let registryToken: string | null = null;
  let registryStoreId: string | null = null;

  // Load existing state
  const saved = loadState();
  if (saved && saved.store_url === config.storeUrl) {
    registryToken = saved.registry_token;
    registryStoreId = saved.store_id;
  }

  return {
    async register_in_registry(input: z.infer<typeof registerInRegistrySchema>) {
      if (!config.registryUrl) throw new Error("ACE_REGISTRY_URL not configured.");

      if (registryToken && registryStoreId) {
        return { store_id: registryStoreId, registered: true, message: "Already registered (reusing token)" };
      }

      const wellKnownUrl = `${config.storeUrl}/.well-known/agent-commerce`;
      const res = await fetch(`${config.registryUrl}/registry/v1/stores`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          well_known_url: wellKnownUrl,
          categories: input.categories || [],
          country: input.country || "US",
        }),
      });

      if (!res.ok) {
        const body = await res.text();
        throw new Error(`Registry error (${res.status}): ${body}`);
      }

      const data = await res.json() as any;
      registryToken = data.registry_token;
      registryStoreId = data.id;

      saveState({
        store_url: config.storeUrl,
        store_id: data.id,
        registry_token: data.registry_token,
        registered_at: new Date().toISOString(),
      });

      return { store_id: data.id, registered: true };
    },

    async sync_products_to_registry(_input: z.infer<typeof syncProductsToRegistrySchema>) {
      if (!config.registryUrl) throw new Error("ACE_REGISTRY_URL not configured.");
      if (!registryToken) throw new Error("Not registered in registry. Run register_in_registry first.");

      // Fetch all products via Admin API
      const productsResp = await client.get(`/api/v1/stores/{store_id}/products`);
      const products = (productsResp.data || productsResp || []).filter((p: any) => p.status === "published");

      // Map to sync format
      const syncPayload = {
        products: products.map((p: any) => {
          const variantPrices = (p.variants || []).map((v: any) => v.price?.amount || 0);
          const allPrices = [p.price?.amount || 0, ...variantPrices];
          return {
            product_id: p.id,
            name: p.name,
            description: p.description || "",
            category: p.category || "general",
            tags: p.tags || [],
            price_range: {
              min: Math.min(...allPrices),
              max: Math.max(...allPrices),
              currency: p.price?.currency || "USD",
            },
            variants_summary: (p.variants || []).map((v: any) => v.name),
            image_url: "",
            in_stock: (p.variants || []).length === 0 || (p.variants || []).some((v: any) => v.inventory > 0),
            rating: { average: 0, count: 0 },
            location: { country: "US", region: "" },
          };
        }),
      };

      const res = await fetch(`${config.registryUrl}/registry/v1/products/sync`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${registryToken}`,
        },
        body: JSON.stringify(syncPayload),
      });

      if (!res.ok) {
        const body = await res.text();
        throw new Error(`Registry sync error (${res.status}): ${body}`);
      }

      const result = await res.json() as any;
      return { synced: result.indexed || 0, errors: result.errors?.length || 0 };
    },
  };
}
```

- [ ] **Step 2: Verify and commit**

```bash
npx tsc --noEmit
git add mcp-seller/src/tools/registry.ts
git commit -m "feat(mcp-seller): add 2 registry tools (register, sync products)"
```

---

### Task 8: MCP Server Entry Point

**Files:**
- Create: `mcp-seller/src/index.ts`

- [ ] **Step 1: Create index.ts**

Wire all 23 tools into the MCP server. Same pattern as mcp-buyer: import McpServer, StdioServerTransport, all tool creators and schemas. Register tools with name, description, schema.shape, and toolHandler wrapper.

Group registrations:
- 8 catalog tools
- 4 order tools
- 5 policy tools
- 4 security tools
- 2 registry tools (conditional on config.registryUrl)

- [ ] **Step 2: Build**

Run: `cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace/mcp-seller" && npm run build`

- [ ] **Step 3: Commit**

```bash
git add mcp-seller/src/index.ts
git commit -m "feat(mcp-seller): add MCP server entry point with all 23 tools"
```

---

### Task 9: README + Final Push

**Files:**
- Create: `mcp-seller/README.md`
- Modify: `README.md` (main repo)

- [ ] **Step 1: Create README**

Cover: what it is, Claude Desktop config, env vars, all 23 tools grouped by domain, example conversation ("publicá todos los productos draft"), development instructions.

- [ ] **Step 2: Update main README**

Add mcp-seller to repo structure.

- [ ] **Step 3: Build, commit, push**

```bash
npm run build
cd "/Users/nicroldan/Desktop/Vibecoding/CLI Marketplace"
git add -A
git commit -m "feat(mcp-seller): complete ACE Seller MCP server v0.1.0"
gh auth switch --user nicoroldan1
git push origin main
gh auth switch --user nicroldan_meli
```
