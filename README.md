<p align="center">
  <h1 align="center">⚡ ACE Protocol</h1>
  <p align="center"><strong>Agent Commerce Exchange</strong></p>
  <p align="center">The open protocol for agent-native commerce — discovery, trust, catalog, and payment-as-auth for the agentic web.</p>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/protocol-ace%2F0.1-6366f1" />
  <img src="https://img.shields.io/badge/license-Apache%202.0-green" />
  <img src="https://img.shields.io/badge/status-alpha-orange" />
  <img src="https://img.shields.io/badge/payments-x402%20%7C%20MPP%20%7C%20Stripe-blue" />
  <img src="https://img.shields.io/badge/built%20with-Go-00ADD8" />
</p>

---

## The Internet Had Its HTTP Moment. Agent Commerce Is Having Its Own.

In the late 1990s, two competing visions of the internet existed: AOL's curated walled garden, and a set of open protocols — HTTP, DNS, HTML. AOL had better UX. Open protocols had no gatekeepers.

We know how that ended.

**Today, the same fork is happening for agent commerce.**

On one side: checkout inside ChatGPT, Gemini, and Copilot. Curated catalogs. Months of BD to get listed. Stringent approval processes. A walled garden with better UX.

On the other side: **Open Agentic Commerce.**

x402 and MPP solved the payment layer — agents can now pay for anything programmatically. But that's only one piece. **The harder problem is everything else:**

- How does an agent *find* what to buy?
- How does a seller *trust* that an autonomous agent won't exceed its authority?
- How does a business *know* what an agent did, and why?

**ACE is the answer to those questions.**

---

## What is ACE?

**ACE (Agent Commerce Exchange)** is an open protocol that defines how autonomous buyer agents discover stores, browse catalogs, manage carts, place orders, and initiate payment — without a human in the loop.

It is **not** a payment protocol. It is the layer that sits *above* payment rails.

```
Agent
  │
  ├─ Queries ACE Registry ──────────── "Find electronics stores in USD that support x402"
  │
  ├─ Reads /.well-known/agent-commerce  "What can this store do? What payment protocols?"
  │
  ├─ Browses catalog                    GET /ace/v1/products?q=wireless+keyboard
  │
  ├─ Creates cart + order               POST /ace/v1/cart → POST /ace/v1/orders
  │
  ├─ Pays via x402 or MPP               POST /ace/v1/orders/{id}/pay
  │
  └─ Immutable audit trail              Every action logged with correlation ID
```

Any agent. Any store. No whitelist. No BD process. Just an open standard.

---

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│                     OPEN SOURCE                            │
│                                                            │
│  ┌─────────────────┐   ┌──────────────────────────────┐   │
│  │   ACE Protocol  │   │   Reference Implementation   │   │
│  │   Specification │   │   (Go — ace-server/)         │   │
│  │   ace-spec/     │   │                              │   │
│  └─────────────────┘   └──────────────────────────────┘   │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Registry (registry/)                                │  │
│  │  · Store registration + crawling                     │  │
│  │  · Agent search by category, country, currency,      │  │
│  │    payment protocol                                  │  │
│  │  · Health monitoring of ACE endpoints                │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Trust Layer                                         │  │
│  │  · Policy engine (ALLOW / DENY / APPROVAL)           │  │
│  │  · Budget limits per agent                           │  │
│  │  · Human-in-the-loop approvals for sensitive actions │  │
│  │  · Immutable audit log with correlation IDs          │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────┐
│                  ANS Cloud (Premium)                       │
│  · Managed ACE hosting — no infrastructure needed          │
│  · MCP Adapter — expose any ACE store to Claude/GPT        │
│  · Verified registry with reputation scoring               │
│  · Advanced ML-based anomaly detection                     │
│  · SLA guarantees                                          │
└────────────────────────────────────────────────────────────┘
```

---

## Protocol at a Glance

### 1. Discovery — `GET /.well-known/agent-commerce`

Every ACE store exposes a machine-readable manifest. No scraping, no guessing.

```json
{
  "store_id": "store_abc123",
  "name": "Acme Widget Store",
  "version": "ace/0.1",
  "ace_base_url": "https://acme-widgets.example.com/ace/v1",
  "capabilities": ["catalog", "cart", "orders", "payments"],
  "currencies": ["USD", "EUR"],
  "payment_protocols": ["x402", "mpp"],
  "auth": { "type": "api_key", "header": "X-ACE-Key" },
  "payment_auth": {
    "enabled": true,
    "header": "X-ACE-Payment",
    "providers": ["x402", "mpp", "mock"],
    "default_currency": "USD"
  },
  "policies_public": {
    "returns": "30-day returns on all items"
  }
}
```

`payment_auth` tells agents they can pay per request without an API key. Just include `X-ACE-Payment: provider:token` in any request.

### 2. Headless Merchant Mode — Payment as Authentication

ACE supports the **headless merchant** model: agents can access any endpoint by paying per request, without accounts or API keys.

```bash
# No API key needed — just pay with the request
curl http://localhost:8081/ace/v1/products \
  -H "X-ACE-Payment: mock:any_token"

# No auth at all? Get a 402 with pricing info
curl http://localhost:8081/ace/v1/products
# → HTTP 402 {"pricing": {"price": 0.00, "currency": "USD", "accepted_providers": ["mock"]}}
```

Every response includes pricing headers so agents always know the cost:
```
X-ACE-Price: 0.00
X-ACE-Currency: USD
```

Both modes work simultaneously — stores accept API keys (for established relationships) and payment tokens (for anonymous, per-request access). The store declares what it supports in `.well-known/agent-commerce`.

### 3. Full Purchase Flow

```
GET  /.well-known/agent-commerce      # Discover store
GET  /ace/v1/products?q=keyboard      # Browse catalog
POST /ace/v1/cart                     # Create cart
POST /ace/v1/cart/{id}/items          # Add items
POST /ace/v1/shipping/quote           # Get shipping options
POST /ace/v1/orders                   # Place order
POST /ace/v1/orders/{id}/pay          # Initiate payment (x402 | mpp | stripe)
GET  /ace/v1/orders/{id}/pay/status   # Poll payment status
```

### 4. Payment Protocol Integration

ACE does not define how payment works — it defines how a store *declares* what it supports and returns the right challenge for each protocol:

| Protocol | Who | Model | ACE response |
|----------|-----|-------|-------------|
| **x402** | Coinbase | Stateless, per-request, on-chain | `{ "type": "x402", "payment_url": "...", "amount": 1500 }` |
| **mpp** | Tempo + Stripe | Stateful session, micropayment streaming | `{ "type": "mpp", "session_endpoint": "...", "amount": 1500 }` |
| **stripe** | Stripe | Legacy fiat | `{ "type": "stripe", "client_secret": "..." }` |
| **mercadopago** | MercadoPago | Legacy LATAM | `{ "type": "mercadopago", "init_point": "..." }` |

### 5. Trust Layer

Sensitive actions require explicit policy configuration. Defaults are safe.

```json
{
  "action": "order.refund",
  "effect": "approval",
  "reason": "Refunds over $50 require human approval"
}
```

Every action is logged immutably:

```json
{
  "id": "audit_789",
  "actor": "agent_buyer_42",
  "actor_type": "agent",
  "action": "order.create",
  "resource": "order_456def",
  "correlation_id": "req_abc123",
  "timestamp": "2026-03-21T14:35:00Z"
}
```

---

## Quick Start

> **Full installation guide** → [GETTING_STARTED.md](GETTING_STARTED.md)
> Covers: local setup, curl walkthrough, connecting Claude/GPT as a buyer agent, deploying your own store.

### Run the Reference Server

```bash
git clone https://github.com/nicoroldan1/agent-commerce-protocol
cd agent-commerce-protocol

# Start the ACE reference server (in-memory, no DB required)
cd ace-server && go run ./cmd/ace-server

# In another terminal, start the registry
cd registry && go run ./cmd/registry
```

Server running at `http://localhost:8080`. Registry at `http://localhost:8081`.

### Run the Demo Buyer Agent

```bash
cd agent-buyer
go run ./cmd/buyer \
  --registry http://localhost:8081 \
  --query "wireless keyboard" \
  --budget 200
```

The demo agent will:
1. Query the registry for matching stores
2. Discover each store via `/.well-known/agent-commerce`
3. Browse the catalog and select a product
4. Create a cart, place an order, and initiate payment
5. Print the full audit trail

### Register Your Store

```bash
curl -X POST http://localhost:8081/registry/v1/stores \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Store",
    "url": "https://my-store.example.com",
    "categories": ["electronics"],
    "country": "US"
  }'
```

---

## Repository Structure

```
agent-commerce-protocol/
│
├── ace-spec/               # Protocol specification (Markdown)
│   └── README.md           # Full ACE v0.1 spec
│
├── ace-server/             # Reference ACE store implementation (Go)
│   ├── cmd/ace-server/     # Entry point
│   └── internal/
│       ├── handlers/       # Buyer API + Admin API handlers
│       ├── middleware/      # Auth (dual mode: API key + payment-as-auth)
│       ├── payment/        # Payment validator interface + mock provider
│       ├── policy/         # Trust layer / policy engine
│       ├── audit/          # Immutable audit logger
│       └── store/          # In-memory data store
│
├── registry/               # Store discovery service (Go)
│   ├── cmd/registry/       # Entry point
│   └── internal/
│       ├── handlers/       # Registry API + Search + Sync handlers
│       ├── search/         # Elasticsearch product search engine
│       ├── auth/           # Registry token generation + validation
│       ├── healthcheck/    # ACE endpoint health monitoring
│       └── store/          # In-memory registry store + token storage
│
├── agent-buyer/            # Demo buyer agent (Go)
│   ├── cmd/buyer/          # Entry point
│   └── internal/client/   # ACE + Registry HTTP clients
│
├── shared/                 # Common types (Go module)
│   └── ace/types.go
│
├── mcp-buyer/              # MCP server for buyer agents (TypeScript)
│   └── src/                # 13 MCP tools: discovery, catalog, purchase
│
├── mcp-seller/             # MCP server for store owners (TypeScript)
│   └── src/                # 23 MCP tools: catalog, orders, policies, security, registry
│
├── ace-connect/            # Platform connectors (TypeScript)
│   └── src/                # Shopify adapter + embedded ACE server
│
└── go.work                 # Go workspace
```

---

## Competitive Landscape

ACE is not competing with payment protocols — it sits above them.

| | ACE | x402 | MPP | AgentCash | ChatGPT Checkout |
|---|:---:|:---:|:---:|:---:|:---:|
| Store discovery | ✅ | ❌ | ❌ | ⚠️ basic | ❌ curated only |
| Product search | ✅ | ❌ | ❌ | ❌ | ✅ |
| Catalog browsing | ✅ | ❌ | ❌ | ❌ | ✅ |
| Cart + orders | ✅ | ❌ | ❌ | ❌ | ✅ |
| Payment-as-auth | ✅ | ✅ | ✅ | ⚠️ | ❌ |
| Per-request pricing | ✅ | ✅ | ✅ | ❌ | ❌ |
| Trust + policies | ✅ | ❌ | ❌ | ❌ | ⚠️ |
| Audit trail | ✅ | ❌ | ❌ | ❌ | ❌ |
| Open protocol | ✅ | ✅ | ✅ | ❌ | ❌ |
| Permissionless | ✅ | ✅ | ✅ | ⚠️ | ❌ |

---

## Roadmap

### Phase 0 — Foundation ✅
- ACE Protocol Specification v0.1
- Reference server (Go) — full buyer + seller admin API
- Registry service — registration, search, health checks
- Demo buyer agent — E2E purchase flow
- In-memory data store

### Phase 0.5 — Scalable Search + Headless Merchant ✅
- Elasticsearch-powered product search across all stores
- Store-to-registry product sync (push model with registry tokens)
- Docker Compose for local development
- **Payment-as-auth** — agents pay per request without accounts (dual mode: API key + payment token)
- **Per-request pricing model** for API-style services
- **Pricing headers** on all buyer API responses (X-ACE-Price, X-ACE-Currency)
- HTTP 402 Payment Required with pricing info for unauthenticated requests

### Phase 1 — Trust + Payment Adapters 🔄
- Production policy engine with configurable rules per store
- Approval workflows (human-in-the-loop for sensitive actions)
- Budget and rate limits per agent
- PostgreSQL persistence
- Real x402 payment adapter (Coinbase / Base / USDC)
- Real MPP payment adapter (Tempo + Stripe)
- Real Stripe payment validation

### Phase 2 — MCP Adapter ✅
- **ACE Buyer MCP** — 13 MCP tools for Claude/GPT to discover, browse, and purchase
- Dual mode: single-store (direct URL) or registry (discovery across all stores)
- TypeScript, published as `ace-buyer-mcp` for `npx` usage

### Phase 3 — Platform Connectors ✅
- **ACE Connect** — connect existing e-commerce stores to ACE with one command
- Adapter pattern: Shopify first, architecture ready for Tiendanube, WooCommerce, CSV
- `npx ace-connect shopify --shop mystore.myshopify.com --token shp_xxx`
- Embeds ACE server, auto-registers in registry, periodic sync

### Phase 4 — ANS Cloud (Managed Platform)
- Hosted ACE endpoints — sellers with no infrastructure
- Managed registry with verification and reputation scoring
- Analytics dashboard
- ML-based anomaly detection
- SLA guarantees

---

## Contributing

ACE is an open protocol. Contributions to the spec, reference implementation, and adapters are welcome.

1. Fork the repo
2. Create a branch: `git checkout -b feat/your-feature`
3. Follow the existing code style (Go, stdlib `net/http`, no heavy frameworks)
4. Open a PR with a clear description of what you're changing and why

For protocol changes (new endpoints, breaking changes), open an issue first to discuss the design.

---

## Design Principles

- **Agent-first** — every endpoint is designed for machine consumption, not humans
- **Payment-agnostic** — ACE does not define how money moves; it defines how commerce *flows*
- **Safety by default** — sensitive actions require explicit policy configuration to allow
- **Least privilege** — agents get the minimum permissions needed for their task
- **Auditability** — every action leaves an immutable trail
- **Open protocol** — anyone can implement ACE; no vendor lock-in, no whitelist

---

## License

Apache 2.0 — see [LICENSE](LICENSE).

---

<p align="center">
  <sub>ACE is the open commerce protocol for the agentic web. Not a walled garden. Not a curated catalog. An open standard — like HTTP, but for agents buying things.</sub>
</p>
