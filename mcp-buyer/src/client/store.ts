import { Config } from "../config.js";

export class StoreClient {
  private config: Config;

  constructor(config: Config) {
    this.config = config;
  }

  private authHeaders(): Record<string, string> {
    if (this.config.apiKey) {
      return { "X-ACE-Key": this.config.apiKey };
    }
    const provider = this.config.paymentProvider;
    const token =
      this.config.paymentToken ||
      (provider === "mock"
        ? `mcp_session_${Math.random().toString(36).slice(2)}`
        : "");
    if (token) {
      return { "X-ACE-Payment": `${provider}:${token}` };
    }
    return {};
  }

  async get(baseUrl: string, path: string, params?: Record<string, string>): Promise<any> {
    const url = new URL(path, baseUrl);
    if (params) {
      Object.entries(params).forEach(([k, v]) => {
        if (v !== undefined && v !== "") url.searchParams.set(k, v);
      });
    }
    const res = await fetch(url.toString(), {
      headers: { ...this.authHeaders(), Accept: "application/json" },
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(
        `ACE API error (${res.status}): ${body.error || res.statusText} [code: ${body.code || "unknown"}]`
      );
    }
    return res.json();
  }

  async post(baseUrl: string, path: string, body?: any): Promise<any> {
    const url = new URL(path, baseUrl);
    const res = await fetch(url.toString(), {
      method: "POST",
      headers: {
        ...this.authHeaders(),
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!res.ok) {
      const respBody = await res.json().catch(() => ({}));
      throw new Error(
        `ACE API error (${res.status}): ${respBody.error || res.statusText} [code: ${respBody.code || "unknown"}]`
      );
    }
    return res.json();
  }

  async discovery(baseUrl: string): Promise<any> {
    const url = new URL("/.well-known/agent-commerce", baseUrl);
    const res = await fetch(url.toString(), {
      headers: { Accept: "application/json" },
    });
    if (!res.ok) throw new Error(`Failed to fetch well-known from ${baseUrl}`);
    return res.json();
  }
}
