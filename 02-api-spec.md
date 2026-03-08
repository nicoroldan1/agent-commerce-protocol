# TAN — API Spec (OpenAPI‑ish) para CloudCode

**Fecha:** 2026-02-22

> Este documento no reemplaza un OpenAPI formal, pero deja listo el contrato para implementarlo.

---

## 1) Convenciones

- Base: `/api/v1` (core) y `/ace/v1` (agent commerce)
- Auth: `Authorization: Bearer <JWT>`
- Idempotencia: `Idempotency-Key: <uuid>`
- Correlation: `X-Request-Id`, `X-Correlation-Id`
- Errores: `{
  "error": {
    "code": "POLICY_APPROVAL_REQUIRED",
    "message": "...",
    "details": {...}
  }
}`

---

## 2) Core Admin API (humano/agente interno)

### 2.1 Productos
- `POST /api/v1/stores/{store_id}/products`
- `GET /api/v1/stores/{store_id}/products?status=&q=&page=`
- `GET /api/v1/stores/{store_id}/products/{product_id}`
- `PATCH /api/v1/stores/{store_id}/products/{product_id}`
- `POST /api/v1/stores/{store_id}/products/{product_id}/publish`
- `POST /api/v1/stores/{store_id}/products/{product_id}/unpublish`

**Policy hooks**:
- create/update: allow
- publish/unpublish: approval default

### 2.2 Inventario
- `PATCH /api/v1/stores/{store_id}/variants/{variant_id}/inventory`
Body:
```json
{ "on_hand": 120, "reason": "sync_supplier_feed" }
```

### 2.3 Órdenes
- `GET /api/v1/stores/{store_id}/orders?status=&from=&to=`
- `GET /api/v1/stores/{store_id}/orders/{order_id}`
- `POST /api/v1/stores/{store_id}/orders/{order_id}/fulfill`
- `POST /api/v1/stores/{store_id}/orders/{order_id}/refund` (approval default)

### 2.4 Policies & Approvals
- `GET /api/v1/stores/{store_id}/policies`
- `PUT /api/v1/stores/{store_id}/policies` (admin only)
- `GET /api/v1/stores/{store_id}/approvals?status=`
- `POST /api/v1/stores/{store_id}/approvals/{approval_id}/approve`
- `POST /api/v1/stores/{store_id}/approvals/{approval_id}/reject`

### 2.5 Auditoría
- `GET /api/v1/stores/{store_id}/audit-logs?actor=&action=&from=&to=`

---

## 3) Agent Commerce Endpoint (ACE)

### 3.1 Descubrimiento
- `GET /.well-known/agent-commerce`

### 3.2 Catálogo
- `GET /ace/v1/products?category=&q=&page=`
- `GET /ace/v1/products/{id}`

### 3.3 Envío
- `POST /ace/v1/shipping/quote`
Body:
```json
{
  "items": [{"variant_id":"v_1","qty":2}],
  "destination": {"country":"AR","postal_code":"1406","city":"CABA"}
}
```

### 3.4 Carrito
- `POST /ace/v1/cart`
- `POST /ace/v1/cart/{cart_id}/items`
- `GET /ace/v1/cart/{cart_id}`

### 3.5 Órdenes
- `POST /ace/v1/orders`
- `GET /ace/v1/orders/{order_id}`

### 3.6 Pago / aprobación
- `POST /ace/v1/orders/{order_id}/pay`
Respuesta posible (approval):
```json
{ "status":"approval_required", "approval_id":"ap_123" }
```

---

## 4) Webhooks (opcional)
- `order.created`
- `order.paid`
- `order.fulfilled`
- `inventory.low`
- `approval.created`
