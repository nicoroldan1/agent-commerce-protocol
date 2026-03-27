export class RegistryClient {
  private baseUrl: string;
  private storeUrlCache: Map<string, string> = new Map();

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  async discoverStores(params: Record<string, string>): Promise<any> {
    const url = new URL("/registry/v1/stores", this.baseUrl);
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== "") url.searchParams.set(k, v);
    });
    const res = await fetch(url.toString(), {
      headers: { Accept: "application/json" },
    });
    if (!res.ok) throw new Error(`Registry error (${res.status}): ${res.statusText}`);
    return res.json();
  }

  async searchProducts(params: Record<string, string>): Promise<any> {
    const url = new URL("/registry/v1/search", this.baseUrl);
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== "") url.searchParams.set(k, v);
    });
    const res = await fetch(url.toString(), {
      headers: { Accept: "application/json" },
    });
    if (!res.ok) throw new Error(`Registry search error (${res.status}): ${res.statusText}`);
    return res.json();
  }

  async getStore(storeId: string): Promise<any> {
    const url = new URL(`/registry/v1/stores/${storeId}`, this.baseUrl);
    const res = await fetch(url.toString(), {
      headers: { Accept: "application/json" },
    });
    if (!res.ok) throw new Error(`Store ${storeId} not found in registry`);
    return res.json();
  }

  async resolveStoreUrl(storeId: string): Promise<string> {
    const cached = this.storeUrlCache.get(storeId);
    if (cached) return cached;

    const store = await this.getStore(storeId);
    const wkUrl = store.well_known_url;
    const res = await fetch(wkUrl, { headers: { Accept: "application/json" } });
    if (!res.ok) throw new Error(`Failed to fetch well-known for store ${storeId}`);
    const wk = await res.json();
    const aceBaseUrl = wk.ace_base_url;
    this.storeUrlCache.set(storeId, aceBaseUrl);
    return aceBaseUrl;
  }
}
