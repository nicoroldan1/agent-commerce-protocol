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
