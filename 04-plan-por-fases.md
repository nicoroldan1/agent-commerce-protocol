# ANS — Development Phases

**Date:** 2026-03-08

---

## Phase 0: Protocol + Registry Foundation (current)

**Goal:** ACE spec published, registry functional, 1-2 demo stores running.

### Deliverables
- ACE Protocol Spec document (markdown) — including `payment_protocols` field in `.well-known/agent-commerce`
- Registry Service (Go) — registration, search, health checks
- ACE Reference Server (Go) — full buyer + seller admin API with payment-agnostic `/pay` endpoint
- Demo Agent Buyer (Go) — E2E purchase flow proof
- Shared types package
- `PaymentAdapter` interface defined (x402, mpp, stripe, mercadopago)

### Exit Criteria
- Demo buyer agent discovers store → browses catalog → creates order → pays
- Registry can register and search stores by payment protocol
- Policy engine blocks sensitive agent actions without approval
- `.well-known/agent-commerce` returns `payment_protocols`

### Status: ✅ Implemented (payment_protocols field pending)

---

## Phase 1: Trust Layer + Payment Protocol Adapters

**Goal:** Production-grade trust and safety + reference payment integrations.

### Deliverables
- Policy engine with configurable rules per store
- Approval workflow (human-in-the-loop for sensitive actions)
- Immutable audit log with correlation IDs
- Budget/rate limits per agent
- Store verification in registry
- PostgreSQL persistence (replace in-memory stores)
- **x402 payment adapter** — reference implementation for on-chain micropayments
- **MPP payment adapter** — reference implementation for session-based streaming payments
- Registry search filterable by `payment_protocols`

### Exit Criteria
- Configurable policies per store
- Budget limits enforced
- Audit log queryable and immutable
- Agent can complete E2E purchase using x402 adapter on testnet

---

## Phase 2: MCP Adapter (Premium)

**Goal:** AI agents can interact with stores natively.

### Deliverables
- MCP server wrapper that exposes any ACE store as tools
- Agent (Claude, etc.) can "connect" to a store and operate natively
- Tool definitions for: browse catalog, add to cart, place order, pay

### Exit Criteria
- Claude can complete a purchase via MCP tools connected to an ACE store

---

## Phase 3: Managed Platform (ANS Cloud)

**Goal:** Full commercial offering.

### Deliverables
- Hosted ACE endpoints (seller doesn't need infrastructure)
- Managed registry with reputation scoring
- Analytics dashboard
- Advanced ML-based anomaly detection
- SLA guarantees

### Exit Criteria
- Paying customers on managed platform
- 99.9% uptime SLA met

---

## Dependencies & Open Decisions

- ~~Payment processor integration~~ — resolved: ACE is payment-agnostic. x402 and MPP are the primary protocols; Stripe/MercadoPago are supported as optional legacy adapters.
- Cloud provider (AWS vs GCP) for managed platform
- Domain strategy (custom domains vs subdomains)
- Federation model for multi-registry
- x402 testnet vs mainnet for Phase 1 adapter (Base/USDC — Coinbase ecosystem)
- MPP integration requires Tempo account or Stripe MPP preview API (`2026-03-04.preview`)
