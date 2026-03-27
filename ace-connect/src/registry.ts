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
