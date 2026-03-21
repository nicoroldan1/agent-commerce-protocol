# ANS — Glossary

- **ANS (Agent Native Store)**: The project — open protocol + registry + premium managed platform for agent commerce.
- **ACE (Agent Commerce Exchange)**: The open protocol specification for agent-to-agent commerce.
- **Registry**: A searchable index of ACE-compatible stores with health monitoring.
- **Trust Layer**: The combination of policies, approvals, audit logs, and budget limits.
- **Policy Engine**: Component that evaluates whether an action is allowed, denied, or requires approval.
- **Approval**: Human-in-the-loop step required for sensitive actions (e.g., publishing products, refunds).
- **Audit Log**: Immutable record of all actions taken in a store, with correlation IDs.
- **MCP Adapter**: Premium wrapper that exposes an ACE store as an MCP server for AI agents.
- **Buyer Agent**: An AI agent that discovers stores and makes purchases via the ACE protocol.
- **Seller Agent**: An AI agent that manages a store (catalog, orders) subject to policies.
- **Well-Known Endpoint**: `GET /.well-known/agent-commerce` — the discovery entry point for any ACE store.
- **Store Entry**: A record in the registry representing a registered ACE-compatible store.
