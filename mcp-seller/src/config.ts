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
