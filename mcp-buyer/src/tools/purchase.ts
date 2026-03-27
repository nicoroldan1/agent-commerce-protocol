import { z } from "zod";
import { StoreClient } from "../client/store.js";
import { Config, resolveStoreUrl } from "../config.js";

export const createCartSchema = z.object({
  store_url: z.string().optional(),
});

export const getCartSchema = z.object({
  store_url: z.string().optional(),
  cart_id: z.string(),
});

export const addToCartSchema = z.object({
  store_url: z.string().optional(),
  cart_id: z.string(),
  product_id: z.string(),
  quantity: z.number(),
  variant_id: z.string().optional(),
});

export const shippingQuoteSchema = z.object({
  store_url: z.string().optional(),
  items: z.array(z.object({
    product_id: z.string(),
    quantity: z.number(),
  })),
  destination: z.object({
    country: z.string(),
    state: z.string().optional(),
    city: z.string().optional(),
    postal_code: z.string().optional(),
  }),
});

export const placeOrderSchema = z.object({
  store_url: z.string().optional(),
  cart_id: z.string(),
});

export const payOrderSchema = z.object({
  store_url: z.string().optional(),
  order_id: z.string(),
  provider: z.string().optional(),
});

export const getOrderSchema = z.object({
  store_url: z.string().optional(),
  order_id: z.string(),
});

export const paymentStatusSchema = z.object({
  store_url: z.string().optional(),
  order_id: z.string(),
});

export function createPurchaseTools(store: StoreClient, config: Config) {
  return {
    async create_cart(input: z.infer<typeof createCartSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      return store.post(baseUrl, "/ace/v1/cart");
    },

    async get_cart(input: z.infer<typeof getCartSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      return store.get(baseUrl, `/ace/v1/cart/${input.cart_id}`);
    },

    async add_to_cart(input: z.infer<typeof addToCartSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      return store.post(baseUrl, `/ace/v1/cart/${input.cart_id}/items`, {
        product_id: input.product_id,
        quantity: input.quantity,
        variant_id: input.variant_id,
      });
    },

    async shipping_quote(input: z.infer<typeof shippingQuoteSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      return store.post(baseUrl, "/ace/v1/shipping/quote", {
        items: input.items,
        destination: input.destination,
      });
    },

    async place_order(input: z.infer<typeof placeOrderSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      return store.post(baseUrl, "/ace/v1/orders", {
        cart_id: input.cart_id,
      });
    },

    async pay_order(input: z.infer<typeof payOrderSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      return store.post(baseUrl, `/ace/v1/orders/${input.order_id}/pay`, {
        provider: input.provider || config.paymentProvider,
      });
    },

    async get_order(input: z.infer<typeof getOrderSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      return store.get(baseUrl, `/ace/v1/orders/${input.order_id}`);
    },

    async payment_status(input: z.infer<typeof paymentStatusSchema>) {
      const baseUrl = resolveStoreUrl(config, input.store_url);
      return store.get(baseUrl, `/ace/v1/orders/${input.order_id}/pay/status`);
    },
  };
}
