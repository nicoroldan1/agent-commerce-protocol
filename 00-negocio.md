# ANS (Agent Native Store) — Business Document

**Date:** 2026-03-08
**Author:** Nicolas Roldan
**Previous name:** TAN (Tienda Agent-Native)

## 1) Executive Summary

**ANS** is an **open agent commerce protocol (ACE)** + **registry** + **premium managed platform**. It follows a hybrid business model like Red Hat: open-source protocol for adoption, managed services as revenue.

The core insight: the future of e-commerce includes **agent-to-agent commerce** — autonomous AI agents buying from and selling to each other. ANS provides the protocol, trust layer, and discovery infrastructure to make this possible.

---

## 2) The Problem

As AI agents become capable of operating tools and making purchasing decisions:

### a) Payment rails exist — but discovery and trust don't
In 2026, two open protocols solve agent payments: **x402** (Coinbase) handles stateless per-request payments via HTTP 402, and **MPP** (Tempo + Stripe) handles stateful session-based micropayment streaming. Both are live and gaining adoption.

What's missing:
- **No standard for store discovery**: agents can't find what to buy
- **No trust layer**: no policies, budget limits, or audit trails for autonomous purchases
- **No catalog protocol**: each integration is custom, one-off, and fragile

### b) Autonomy without controls is risky
- Agents making purchases need guardrails: budget limits, approval workflows
- Fraud/abuse risk increases without policy enforcement
- Compliance and accountability require immutable audit trails

---

## 3) Value Proposition

### For the ecosystem (open source)
- **ACE Protocol**: universal standard for agent **discovery + catalog + trust** — payment-agnostic, works with x402, MPP, Stripe, or any future protocol
- **Registry**: decentralized discovery — agents find stores via a searchable index
- **Trust Layer**: policies, approvals, budget limits, and immutable audit logs

### For businesses (premium — ANS Cloud)
- **Managed hosting**: run ACE-compatible stores without infrastructure
- **MCP Adapter**: expose any ACE store as an MCP server for AI agents (Claude, etc.)
- **Verification & reputation**: managed registry with trust signals
- **Analytics & ML**: anomaly detection, advanced policies
- **SLA guarantees**

---

## 4) What We Build

### Open Source (ACE Protocol + Reference Implementation)

1. **ACE Protocol Specification**
   A standard for agent commerce: discovery via `.well-known/agent-commerce`, catalog browsing, cart management, ordering, and payment.

2. **Reference Server**
   A Go backend implementing the full ACE protocol — catalog, cart, orders, payments, policies, audit.

3. **Registry**
   A searchable index of ACE-compatible stores with health monitoring.

4. **Trust Layer**
   - Policies (ALLOW/DENY/APPROVAL) per action
   - Immutable audit log with correlation IDs
   - Budget limits per agent

### Premium (ANS Cloud)

- Managed ACE endpoints (no infrastructure needed)
- MCP Adapter (wrap ACE stores as MCP servers)
- Managed registry with verification and reputation scoring
- Advanced ML-based anomaly detection
- Analytics dashboard
- SLA guarantees

---

## 5) How Discovery Works

**Model:** Registry as index (like Google for stores)

1. Stores expose `GET /.well-known/agent-commerce` with their capabilities
2. Registry crawls and indexes registered stores
3. Buyer agents query the registry to find stores by category, country, currency
4. Buyer agents connect directly to stores via the ACE protocol

---

## 6) Business Model

### Hybrid (Red Hat model)

**Open source (adoption):**
- ACE protocol spec — free, open standard
- Reference implementation — free, self-hostable
- Registry — free, self-hostable

**Premium (revenue):**
- ANS Cloud managed hosting — subscription per store
- MCP Adapter — per-connection or subscription
- Managed registry — verification fees, premium listings
- Enterprise features — advanced policies, ML anomaly detection, SLA

---

## 7) Design Principles

- **No human storefront**: value is in the agent-native layer
- **Agent-first**: every endpoint designed for machine consumption
- **Safety by default**: sensitive actions require approval
- **Least privilege**: minimal permissions for each agent
- **Auditability**: every action is logged immutably
- **Open protocol**: anyone can implement ACE; no vendor lock-in

---

## 8) What ANS Is NOT

- Not a marketplace for selling agents
- Not a Shopify clone (no human storefront)
- Not targeting current human sellers (targeting the future of agent-to-agent commerce)
- Not a payment protocol — ACE is payment-agnostic; stores declare their supported payment protocols (x402, MPP, Stripe, etc.) and agents negotiate payment directly using those standards

---

## 9) Competitive Landscape

| Player | What they solve | What they don't |
|--------|----------------|-----------------|
| **x402** (Coinbase) | Stateless per-request payments via HTTP 402 | Discovery, trust, catalog |
| **MPP** (Tempo + Stripe) | Session-based micropayment streaming, multi-rail | Discovery, trust, catalog |
| **AP2** (Google) | HTTP 402-based payments | Discovery, trust, catalog |
| **AgentCash** | Payment + basic merchant discovery (bundled) | Trust layer, policies, open protocol |
| **Checkout in ChatGPT/Gemini** | Curated catalog + checkout UX | Open, permissionless, trust for autonomous agents |
| **ANS / ACE** | Discovery + trust + catalog (payment-agnostic) | Is not a payment rail — delegates to x402/MPP |

**Positioning:** ANS/ACE is the layer that sits above payment rails. Where x402 and MPP answer "how does an agent pay?", ACE answers "how does an agent find what to buy, verify the seller, and operate safely within guardrails?"

---

## 10) Quick Glossary

- **ANS**: Agent Native Store — the project and premium platform
- **ACE**: Agent Commerce Exchange — the open protocol (discovery + trust + catalog, payment-agnostic)
- **Registry**: searchable index of ACE-compatible stores
- **Policy**: rules that control what actions agents can take
- **Approval**: human-in-the-loop step for sensitive actions
- **MCP Adapter**: wrapper that exposes ACE stores as MCP servers
- **x402**: Coinbase's open payment protocol — stateless HTTP 402-based per-request payments
- **MPP**: Machine Payments Protocol by Tempo + Stripe — session-based micropayment streaming, multi-rail
- **payment_protocols**: field in `.well-known/agent-commerce` declaring which payment standards a store supports
