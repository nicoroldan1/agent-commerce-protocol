# ANS — API Specification Summary

**Date:** 2026-03-08

> For the full protocol specification, see `ace-spec/README.md`.

---

## 1) Conventions

- Buyer API base: `/ace/v1`
- Admin API base: `/api/v1`
- Auth (buyer): `X-ACE-Key: <api-key>`
- Auth (admin): `Authorization: Bearer <token>`
- Errors: `{ "error": "message", "code": "CODE", "details": "..." }`
- Pagination: `?offset=0&limit=20` → `{ "data": [...], "total": N, "offset": N, "limit": N }`

---

## 2) Buyer API (ACE Protocol — public)

### Discovery
- `GET /.well-known/agent-commerce` → Store info, capabilities, auth config, supported payment protocols

  **Response includes:**
  ```json
  {
    "store_id": "...",
    "name": "...",
    "ace_base_url": "...",
    "capabilities": ["catalog", "cart", "orders", "payments"],
    "auth": { "type": "api_key", "header": "X-ACE-Key" },
    "currencies": ["USD", "EUR"],
    "payment_protocols": ["x402", "mpp"],
    "policies": {}
  }
  ```

### Catalog
- `GET /ace/v1/products?q=&category=&offset=&limit=` → Paginated product list
- `GET /ace/v1/products/{id}` → Product detail

### Shipping
- `POST /ace/v1/shipping/quote` → Shipping options (body: items + destination)

### Cart
- `POST /ace/v1/cart` → Create empty cart
- `POST /ace/v1/cart/{id}/items` → Add item (body: product_id, variant_id, quantity)
- `GET /ace/v1/cart/{id}` → Cart with items and total

### Orders
- `POST /ace/v1/orders` → Create order from cart (body: cart_id)
- `GET /ace/v1/orders/{id}` → Order status

### Payments
- `POST /ace/v1/orders/{id}/pay` → Initiate payment

  **Request body:**
  ```json
  { "protocol": "x402" | "mpp" | "stripe" | "mercadopago" }
  ```

  **Response (protocol-specific):**

  `x402`:
  ```json
  { "type": "x402", "payment_url": "...", "amount": 1500, "currency": "USD", "expires_at": "..." }
  ```
  Agent submits on-chain payment and retries request with `X-PAYMENT: <proof>` header.

  `mpp`:
  ```json
  { "type": "mpp", "session_endpoint": "...", "amount": 1500, "currency": "USD" }
  ```
  Agent opens MPP session and streams micropayments until amount is met.

  `stripe` / `mercadopago`:
  ```json
  { "type": "stripe", "client_secret": "...", "amount": 1500, "currency": "USD" }
  ```

- `GET /ace/v1/orders/{id}/pay/status` → Payment status (works for all protocols)

---

## 3) Seller Admin API (private)

### Catalog Management
- `POST /api/v1/stores/{store_id}/products` → Create product (draft)
- `GET /api/v1/stores/{store_id}/products` → List products
- `PATCH /api/v1/stores/{store_id}/products/{id}` → Update product
- `DELETE /api/v1/stores/{store_id}/products/{id}` → Delete product
- `POST /api/v1/stores/{store_id}/products/{id}/publish` → Publish (policy check)
- `POST /api/v1/stores/{store_id}/products/{id}/unpublish` → Unpublish

### Inventory
- `PATCH /api/v1/stores/{store_id}/variants/{id}/inventory` → Update stock

### Orders
- `GET /api/v1/stores/{store_id}/orders` → List orders
- `GET /api/v1/stores/{store_id}/orders/{id}` → Order detail
- `POST /api/v1/stores/{store_id}/orders/{id}/fulfill` → Mark fulfilled
- `POST /api/v1/stores/{store_id}/orders/{id}/refund` → Refund (policy check)

### Policies & Approvals
- `GET /api/v1/stores/{store_id}/policies` → View policies
- `PUT /api/v1/stores/{store_id}/policies` → Update policies
- `GET /api/v1/stores/{store_id}/approvals` → List pending approvals
- `POST /api/v1/stores/{store_id}/approvals/{id}/approve` → Approve
- `POST /api/v1/stores/{store_id}/approvals/{id}/reject` → Reject

### Audit & API Keys
- `GET /api/v1/stores/{store_id}/audit-logs` → Query audit trail
- `POST /api/v1/stores/{store_id}/api-keys` → Create buyer API key
- `GET /api/v1/stores/{store_id}/api-keys` → List keys
- `DELETE /api/v1/stores/{store_id}/api-keys/{id}` → Revoke key

---

## 4) Registry API (public)

- `POST /registry/v1/stores` → Register store (body: well_known_url, categories, country)
- `GET /registry/v1/stores?q=&category=&country=&currency=` → Search stores
- `GET /registry/v1/stores/{id}` → Store detail + health
- `GET /registry/v1/stores/{id}/health` → Trigger health check
