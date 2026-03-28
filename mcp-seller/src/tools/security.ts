import { z } from "zod";
import { AdminClient } from "../client.js";
import { Config } from "../config.js";

export const listApiKeysSchema = z.object({});
export const createApiKeySchema = z.object({
  name: z.string(),
  scopes: z.array(z.string()),
});
export const deleteApiKeySchema = z.object({ key_id: z.string() });
export const listAuditLogsSchema = z.object({
  action: z.string().optional(),
  actor: z.string().optional(),
  offset: z.number().optional(),
  limit: z.number().optional(),
});

export function createSecurityTools(client: AdminClient, config: Config) {
  const base = `/api/v1/stores/{store_id}`;

  return {
    async list_api_keys(_input: z.infer<typeof listApiKeysSchema>) {
      return client.get(`${base}/api-keys`);
    },

    async create_api_key(input: z.infer<typeof createApiKeySchema>) {
      return client.post(`${base}/api-keys`, input);
    },

    async delete_api_key(input: z.infer<typeof deleteApiKeySchema>) {
      await client.delete(`${base}/api-keys/${input.key_id}`);
      return { deleted: true };
    },

    async list_audit_logs(input: z.infer<typeof listAuditLogsSchema>) {
      const params: Record<string, string> = {};
      if (input.action) params.action = input.action;
      if (input.actor) params.actor = input.actor;
      if (input.offset !== undefined) params.offset = String(input.offset);
      if (input.limit !== undefined) params.limit = String(input.limit);
      return client.get(`${base}/audit-logs`, params);
    },
  };
}
