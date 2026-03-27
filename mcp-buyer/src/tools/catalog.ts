import { z } from "zod";
import { StoreClient } from "../client/store.js";
import { Config, resolveStoreUrl } from "../config.js";

export const browseStoreSchema = z.object({
  store_url: z.string().optional(),
  category: z.string().optional(),
  query: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

export const getProductSchema = z.object({
  store_url: z.string().optional(),
  product_id: z.string(),
});

export const getPricingSchema = z.object({
  store_url: z.string().optional(),
});

export function createCatalogTools(store: StoreClient, config: Config) {
  return {
    async browse_store(input: z.infer<typeof browseStoreSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      const params: Record<string, string> = {};
      if (input.category) params.category = input.category;
      if (input.query) params.q = input.query;
      if (input.offset !== undefined) params.offset = String(input.offset);
      if (input.limit !== undefined) params.limit = String(input.limit);
      return store.get(baseUrl, "/ace/v1/products", params);
    },

    async get_product(input: z.infer<typeof getProductSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      return store.get(baseUrl, `/ace/v1/products/${input.product_id}`);
    },

    async get_pricing(input: z.infer<typeof getPricingSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      // Pricing endpoint is public, no auth needed, but we use the same client for simplicity
      return store.get(baseUrl, "/ace/v1/pricing");
    },
  };
}
