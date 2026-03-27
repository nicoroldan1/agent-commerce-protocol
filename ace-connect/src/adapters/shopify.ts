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
