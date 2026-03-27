export interface Config {
  registryUrl: string | null;
  storeUrl: string | null;
  apiKey: string | null;
  paymentProvider: string;
  paymentToken: string | null;
}

export function loadConfig(): Config {
  const registryUrl = process.env.ACE_REGISTRY_URL || null;
  const storeUrl = process.env.ACE_STORE_URL || null;

  if (!registryUrl && !storeUrl) {
    throw new Error(
      "At least one of ACE_REGISTRY_URL or ACE_STORE_URL must be set."
    );
  }

  return {
    registryUrl,
    storeUrl,
    apiKey: process.env.ACE_API_KEY || null,
    paymentProvider: process.env.ACE_PAYMENT_PROVIDER || "mock",
    paymentToken: process.env.ACE_PAYMENT_TOKEN || null,
  };
}

export function resolveStoreUrl(config: Config, storeUrl?: string): string {
  const url = storeUrl || config.storeUrl;
  if (!url) {
    throw new Error(
      "No store URL provided. Set ACE_STORE_URL or pass store_url parameter. Use discover_stores to find stores."
    );
  }
  return url;
}
