# ANS — Technical Specification

**Date:** 2026-03-08
**Previous name:** TAN (Tienda Agent-Native)

---

## 0) Scope & Assumptions

### Scope
- Open ACE (Agent Commerce Exchange) protocol specification
- Registry service for store discovery and health monitoring
- Reference ACE server implementation (Go)
- Trust layer: policies, approvals, audit logging
- Demo buyer agent for E2E validation

### Out of Scope (for now)
- Human storefront / UI
- MCP Adapter (Phase 2 — premium)
- Managed hosting (Phase 3 — premium)
- ML-based anomaly detection (Phase 3)

### Stack
- **Language:** Go (stdlib net/http)
- **Database:** PostgreSQL (production), in-memory (MVP)
- **Payments:** Payment-agnostic — stores declare supported protocols via `payment_protocols` in `.well-known/agent-commerce`. Reference adapters provided for x402 (Coinbase) and MPP (Tempo + Stripe). Stripe and MercadoPago supported as legacy/fallback adapters.
- **Hosting:** AWS (ECS/Lambda) or GCP (Cloud Run)
- **CI/CD:** GitHub Actions
- **Protocol spec:** Markdown + OpenAPI

---

## 1) Architecture

```
┌─────────────────────────────────────────────────────┐
│                    OPEN SOURCE                       │
│                                                      │
│  ┌──────────────┐    ┌──────────────────────────┐   │
│  │  ACE Spec    │    │  Reference Implementation │   │
│  │  (Protocol)  │    │  (Go backend + demo store)│   │
│  └──────────────┘    └──────────────────────────┘   │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │  Registry (self-hostable)                     │   │
│  │  - Store registration + indexing              │   │
│  │  - Agent search/discovery                     │   │
│  │  - Health checks on ACE endpoints             │   │
│  └──────────────────────────────────────────────┘   │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │  Trust Layer v0                               │   │
│  │  - Policies (ALLOW/DENY/APPROVAL)             │   │
│  │  - Audit log (immutable)                      │   │
│  │  - Budget limits per agent                    │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│                    PREMIUM (ANS Cloud)                │
│                                                      │
│  - Managed registry with verification & reputation   │
│  - MCP Adapter (expose ACE stores as MCP servers)    │
│  - Managed hosting for ACE endpoints                 │
│  - Advanced policies + ML anomaly detection          │
│  - Analytics dashboard                               │
│  - SLA guarantees                                    │
└─────────────────────────────────────────────────────┘
```

### Components

| Component | Description | Location |
|-----------|-------------|----------|
| ACE Spec | Protocol specification (markdown) | `ace-spec/` |
| Registry | Store discovery + health checks (Go) | `registry/` |
| ACE Server | Reference store implementation (Go) | `ace-server/` |
| Agent Buyer | Demo CLI buyer agent (Go) | `agent-buyer/` |
| Shared | Common types and helpers | `shared/` |

---

## 2) ACE Protocol Summary

### Discovery
Every ACE store exposes `GET /.well-known/agent-commerce` returning:
- Store ID, name, version
- ACE base URL
- Capabilities (catalog, cart, orders, payments)
- Auth configuration
- Supported currencies
- Public policies
- **`payment_protocols`**: list of supported payment standards (e.g. `["x402", "mpp", "stripe"]`). Agents use this to select the payment method they support. ACE does not mandate a specific protocol.

### Authentication (v0.1)
- API Keys via `X-ACE-Key` header
- Keys issued by store owner to buyer agents
- Future: OAuth2, federation

### Buyer API (public, /ace/v1/)
Full purchase flow:
- Product discovery and search
- Shipping quotes
- Cart management
- Order creation
- Payment initiation — protocol-agnostic; client specifies `protocol` (x402 | mpp | stripe | mercadopago); server responds with the appropriate challenge/instructions for that protocol

### Seller Admin API (private, /api/v1/)
Store management:
- Catalog CRUD + publish/unpublish
- Inventory management
- Order fulfillment and refunds
- Policy configuration
- Approval workflows
- Audit log queries
- API key management

See `ace-spec/README.md` for full protocol specification.

---

## 3) Trust Layer

### Policy Engine
- Configurable per-store rules
- Action-based: each operation type has a policy (allow/deny/approval)
- Default: agents need approval for `product.publish` and `order.refund`
- Humans can bypass policies; agents cannot

### Audit Log
- Immutable append-only log
- Correlation IDs for tracing
- Records: actor, action, resource, timestamp, details
- Actor types: human, agent

### Approvals
- Human-in-the-loop for sensitive actions
- Pending queue with approve/reject workflow
- Triggered automatically by policy engine

---

## 4) Data Model (in-memory for MVP)

### Core Entities
- **Product**: id, name, description, price, variants, status (draft/published/unpublished)
- **Variant**: id, name, SKU, price, inventory, attributes
- **Cart**: id, items, total
- **Order**: id, cart_id, items, total, status (pending/paid/fulfilled/refunded/cancelled)
- **Payment**: id, order_id, status, provider, amount, external_id

### Trust Entities
- **Policy**: id, action, effect (allow/deny/approval)
- **Approval**: id, action, resource, status (pending/approved/rejected)
- **AuditEntry**: id, store_id, action, actor, actor_type, resource, correlation_id, timestamp

### Registry Entities
- **StoreEntry**: id, well_known_url, name, categories, country, currencies, capabilities, health_status

---

## 5) Auth Model

| Actor | Auth Method | Policy Bypass |
|-------|------------|---------------|
| Human admin | JWT (email/password or OAuth) | Yes |
| Seller agent | API key with scoped permissions | No (goes through policy engine) |
| Buyer agent | API key via X-ACE-Key header | N/A (read + purchase only) |

---

## 6) Error Handling

Standard error format across all APIs:
```json
{
  "error": "human-readable message",
  "code": "MACHINE_CODE",
  "details": "optional extra info"
}
```

Codes: NOT_FOUND, INVALID_REQUEST, UNAUTHORIZED, OUT_OF_STOCK, PAYMENT_FAILED, POLICY_DENIED, APPROVAL_REQUIRED, INTERNAL_ERROR

---

## 7) Payment Protocol Support

ACE is payment-agnostic. Each store declares which protocols it accepts in `.well-known/agent-commerce`. The payment step in the buyer flow works as follows:

1. Agent reads `payment_protocols` from `.well-known/agent-commerce`
2. Agent selects a protocol it supports
3. Agent calls `POST /ace/v1/orders/{id}/pay` with `{ "protocol": "x402" | "mpp" | "stripe" | ... }`
4. ACE server responds with protocol-specific instructions:

| Protocol | ACE server response |
|----------|-------------------|
| **x402** | Returns `{ "type": "x402", "payment_url": "...", "amount": ..., "currency": ... }` — agent submits payment on-chain and retries with `X-PAYMENT` header |
| **mpp** | Returns `{ "type": "mpp", "session_endpoint": "...", "amount": ..., "currency": ... }` — agent opens MPP session and streams micropayments |
| **stripe** | Returns `{ "type": "stripe", "client_secret": "...", "amount": ..., "currency": ... }` — legacy flow for human-assisted or card payments |
| **mercadopago** | Returns `{ "type": "mercadopago", "init_point": "...", "amount": ..., "currency": ... }` — legacy flow for LATAM |

Reference adapters for x402 and MPP are provided in `ace-server/payment/`. Adding new protocol adapters requires implementing the `PaymentAdapter` interface.

---

## 8) Testing Strategy

- **Protocol compliance**: demo buyer agent runs full E2E flow
- **Registry**: register → search → health check
- **Trust layer**: attempt action exceeding policy → verify denial/approval
- **Payment protocols**: E2E test with x402 adapter (testnet) and MPP adapter (dry-run mode)
- **Future**: MCP adapter test with Claude
