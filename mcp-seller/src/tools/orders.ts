import { z } from "zod";
import { AdminClient } from "../client.js";
import { Config } from "../config.js";

export const listOrdersSchema = z.object({
  offset: z.number().optional(),
  limit: z.number().optional(),
});
export const getOrderSchema = z.object({ order_id: z.string() });
export const fulfillOrderSchema = z.object({ order_id: z.string() });
export const refundOrderSchema = z.object({ order_id: z.string() });

export function createOrderTools(client: AdminClient, config: Config) {
  const base = `/api/v1/stores/{store_id}`;

  return {
    async list_orders(input: z.infer<typeof listOrdersSchema>) {
      const params: Record<string, string> = {};
      if (input.offset !== undefined) params.offset = String(input.offset);
      if (input.limit !== undefined) params.limit = String(input.limit);
      return client.get(`${base}/orders`, params);
    },

    async get_order(input: z.infer<typeof getOrderSchema>) {
      return client.get(`${base}/orders/${input.order_id}`);
    },

    async fulfill_order(input: z.infer<typeof fulfillOrderSchema>) {
      return client.post(`${base}/orders/${input.order_id}/fulfill`);
    },

    async refund_order(input: z.infer<typeof refundOrderSchema>) {
      return client.post(`${base}/orders/${input.order_id}/refund`);
    },
  };
}
