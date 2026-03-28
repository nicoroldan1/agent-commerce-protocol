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
