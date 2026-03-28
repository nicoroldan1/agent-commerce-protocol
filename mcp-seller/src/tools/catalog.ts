import { z } from "zod";
import { AdminClient } from "../client.js";
import { Config } from "../config.js";

const variantSchema = z.object({
  name: z.string(),
  sku: z.string().optional(),
  price: z.object({ amount: z.number(), currency: z.string() }),
  inventory: z.number(),
  attributes: z.record(z.string()).optional(),
});

export const listProductsSchema = z.object({
  offset: z.number().optional(),
  limit: z.number().optional(),
});

export const createProductSchema = z.object({
  name: z.string(),
  description: z.string(),
  price: z.object({ amount: z.number(), currency: z.string() }),
  variants: z.array(variantSchema).optional(),
});

export const bulkCreateProductsSchema = z.object({
  products: z.array(z.object({
    name: z.string(),
    description: z.string(),
    price: z.object({ amount: z.number(), currency: z.string() }),
    variants: z.array(variantSchema).optional(),
  })),
});

export const updateProductSchema = z.object({
  product_id: z.string(),
  name: z.string().optional(),
  description: z.string().optional(),
  price: z.object({ amount: z.number(), currency: z.string() }).optional(),
});

export const deleteProductSchema = z.object({ product_id: z.string() });
export const publishProductSchema = z.object({ product_id: z.string() });
export const unpublishProductSchema = z.object({ product_id: z.string() });

export const updateInventorySchema = z.object({
  variant_id: z.string(),
  inventory: z.number(),
});

export function createCatalogTools(client: AdminClient, config: Config) {
  const base = `/api/v1/stores/{store_id}`;

  return {
    async list_products(input: z.infer<typeof listProductsSchema>) {
      const params: Record<string, string> = {};
      if (input.offset !== undefined) params.offset = String(input.offset);
      if (input.limit !== undefined) params.limit = String(input.limit);
      return client.get(`${base}/products`, params);
    },

    async create_product(input: z.infer<typeof createProductSchema>) {
      return client.post(`${base}/products`, input);
    },

    async bulk_create_products(input: z.infer<typeof bulkCreateProductsSchema>) {
      const result = { created: 0, errors: [] as Array<{ index: number; name: string; error: string }> };
      for (let i = 0; i < input.products.length; i++) {
        try {
          await client.post(`${base}/products`, input.products[i]);
          result.created++;
        } catch (err: any) {
          result.errors.push({ index: i, name: input.products[i].name, error: err.message });
        }
      }
      return result;
    },

    async update_product(input: z.infer<typeof updateProductSchema>) {
      const { product_id, ...updates } = input;
      return client.patch(`${base}/products/${product_id}`, updates);
    },

    async delete_product(input: z.infer<typeof deleteProductSchema>) {
      await client.delete(`${base}/products/${input.product_id}`);
      return { deleted: true };
    },

    async publish_product(input: z.infer<typeof publishProductSchema>) {
      return client.post(`${base}/products/${input.product_id}/publish`);
    },

    async unpublish_product(input: z.infer<typeof unpublishProductSchema>) {
      return client.post(`${base}/products/${input.product_id}/unpublish`);
    },

    async update_inventory(input: z.infer<typeof updateInventorySchema>) {
      await client.patch(`${base}/variants/${input.variant_id}/inventory`, { inventory: input.inventory });
      return { updated: true };
    },
  };
}
