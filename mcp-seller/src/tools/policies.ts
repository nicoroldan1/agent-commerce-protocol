import { z } from "zod";
import { AdminClient } from "../client.js";
import { Config } from "../config.js";

export const getPoliciesSchema = z.object({});
export const updatePoliciesSchema = z.object({
  policies: z.array(z.object({
    action: z.string(),
    effect: z.enum(["allow", "deny", "approval"]),
  })),
});
export const listApprovalsSchema = z.object({});
export const approveActionSchema = z.object({ approval_id: z.string() });
export const rejectActionSchema = z.object({ approval_id: z.string() });

export function createPolicyTools(client: AdminClient, config: Config) {
  const base = `/api/v1/stores/{store_id}`;

  return {
    async get_policies(_input: z.infer<typeof getPoliciesSchema>) {
      return client.get(`${base}/policies`);
    },

    async update_policies(input: z.infer<typeof updatePoliciesSchema>) {
      return client.put(`${base}/policies`, input.policies);
    },

    async list_approvals(_input: z.infer<typeof listApprovalsSchema>) {
      return client.get(`${base}/approvals`);
    },

    async approve_action(input: z.infer<typeof approveActionSchema>) {
      return client.post(`${base}/approvals/${input.approval_id}/approve`);
    },

    async reject_action(input: z.infer<typeof rejectActionSchema>) {
      return client.post(`${base}/approvals/${input.approval_id}/reject`);
    },
  };
}
