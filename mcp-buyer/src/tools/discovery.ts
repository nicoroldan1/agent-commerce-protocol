import { z } from "zod";
import { RegistryClient } from "../client/registry.js";

export const discoverStoresSchema = z.object({
  query: z.string().optional(),
  category: z.string().optional(),
  country: z.string().optional(),
  currency: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

export const searchProductsSchema = z.object({
  query: z.string(),
  category: z.string().optional(),
  country: z.string().optional(),
  currency: z.string().optional(),
  price_min: z.number().optional(),
  price_max: z.number().optional(),
  in_stock: z.boolean().optional(),
  sort: z.enum(["relevance", "price_asc", "price_desc", "rating"]).optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

export function createDiscoveryTools(registry: RegistryClient | null) {
  return {
    async discover_stores(input: z.infer<typeof discoverStoresSchema>) {
      if (!registry) throw new Error("ACE_REGISTRY_URL not configured. Cannot discover stores.");
      const params: Record<string, string> = {};
      if (input.query) params.q = input.query;
      if (input.category) params.category = input.category;
      if (input.country) params.country = input.country;
      if (input.currency) params.currency = input.currency;
      if (input.offset !== undefined) params.offset = String(input.offset);
      if (input.limit !== undefined) params.limit = String(input.limit);
      return registry.discoverStores(params);
    },

    async search_products(input: z.infer<typeof searchProductsSchema>) {
      if (!registry) throw new Error("ACE_REGISTRY_URL not configured. Cannot search products.");
      const params: Record<string, string> = { q: input.query };
      if (input.category) params.category = input.category;
      if (input.country) params.country = input.country;
      if (input.currency) params.currency = input.currency;
      if (input.price_min !== undefined) params.price_min = String(input.price_min);
      if (input.price_max !== undefined) params.price_max = String(input.price_max);
      if (input.in_stock !== undefined) params.in_stock = String(input.in_stock);
      if (input.sort) params.sort = input.sort;
      if (input.offset !== undefined) params.offset = String(input.offset);
      if (input.limit !== undefined) params.limit = String(input.limit);
      return registry.searchProducts(params);
    },
  };
}
