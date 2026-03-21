# ACE Protocol Specification

**Agent Commerce Exchange (ACE) Protocol v0.1**

---

## Table of Contents

1. [Overview](#1-overview)
2. [Discovery](#2-discovery)
3. [Authentication](#3-authentication)
4. [Buyer API Endpoints](#4-buyer-api-endpoints)
5. [Error Format](#5-error-format)
6. [Pagination](#6-pagination)
7. [Versioning](#7-versioning)
8. [Seller Admin API (Informational)](#8-seller-admin-api-informational)
9. [Registry API (Informational)](#9-registry-api-informational)

---

## 1. Overview

### What is ACE?

ACE (Agent Commerce Exchange) is an open protocol that defines how autonomous buyer agents discover and transact with online stores. It provides a standardized interface for the entire purchase lifecycle: product discovery, cart management, order placement, and payment initiation.

ACE is the public-facing protocol of the ANS (Agent Network for Shopping) ecosystem. Any store that implements ACE can be discovered and used by any ACE-compatible buyer agent.

### Design Principles

- **REST + JSON.** All endpoints accept and return JSON over HTTPS. No HTML, no XML, no server-rendered UI.
- **Stateless.** Every request carries all the information needed to process it. Servers do not rely on session state between requests.
- **Agent-first.** The protocol is designed for machine-to-machine communication. There is no assumption of a human UI. Responses are structured for programmatic consumption.
- **Minimal surface.** The protocol defines the smallest useful set of operations. Stores MAY extend with additional capabilities, but the core spec is intentionally lean.

### Version

This document specifies **ace/0.1** -- the first draft of the protocol.

---

## 2. Discovery

Every ACE-compatible store MUST expose a discovery endpoint at a well-known URL.

### Endpoint

```
GET /.well-known/agent-commerce
```

This endpoint requires no authentication. It is the entry point for any buyer agent to learn about the store's capabilities and how to interact with it.

### Response

**Status:** `200 OK`

**Content-Type:** `application/json`

```json
{
  "store_id": "store_abc123",
  "name": "Acme Widget Store",
  "version": "ace/0.1",
  "ace_base_url": "https://acme-widgets.example.com/ace/v1",
  "capabilities": ["catalog", "cart", "orders", "payments"],
  "auth": {
    "type": "api_key",
    "header": "X-ACE-Key"
  },
  "currencies": ["USD"],
  "policies_public": {
    "returns": "30-day returns on all items",
    "shipping": "Ships to US and Canada"
  }
}
```

### Field Definitions

| Field | Type | Required | Description |
|---|---|---|---|
| `store_id` | string | Yes | Unique identifier for this store. |
| `name` | string | Yes | Human-readable store name. |
| `version` | string | Yes | ACE protocol version. MUST be `"ace/0.1"` for this spec. |
| `ace_base_url` | string | Yes | Base URL for all Buyer API endpoints. All paths in Section 4 are relative to this URL. |
| `capabilities` | string[] | Yes | List of supported capability groups: `"catalog"`, `"cart"`, `"orders"`, `"payments"`. |
| `auth` | object | Yes | Authentication configuration. See Section 3. |
| `auth.type` | string | Yes | Authentication method. MUST be `"api_key"` in v0.1. |
| `auth.header` | string | Yes | HTTP header name for the API key. MUST be `"X-ACE-Key"` in v0.1. |
| `currencies` | string[] | Yes | ISO 4217 currency codes accepted by this store. |
| `policies_public` | object | No | Free-form object with public policy information (returns, shipping, etc.). |

---

## 3. Authentication

### v0.1: API Key

In ACE v0.1, authentication is performed via an API key passed in an HTTP header.

- The header name is `X-ACE-Key`.
- The key is provided by the store owner to authorized buyer agents out of band (e.g., via a seller admin dashboard or manual exchange).
- Every request to a Buyer API endpoint (Section 4) MUST include this header.

**Example request header:**

```
X-ACE-Key: sk_live_abc123def456
```

If the key is missing or invalid, the store MUST respond with:

**Status:** `401 Unauthorized`

```json
{
  "error": "Invalid or missing API key",
  "code": "UNAUTHORIZED"
}
```

### Future Versions

Future versions of ACE may support additional authentication methods, including:

- OAuth 2.0 client credentials flow
- Federated identity via the ANS registry

These are not part of the v0.1 specification.

---

## 4. Buyer API Endpoints

All endpoints in this section are relative to the `ace_base_url` returned by the discovery endpoint. For example, if `ace_base_url` is `https://acme-widgets.example.com/ace/v1`, then the products endpoint is `https://acme-widgets.example.com/ace/v1/products`.

All endpoints require authentication (see Section 3) unless otherwise noted.

All request bodies MUST be sent as `Content-Type: application/json`.

All responses are `Content-Type: application/json`.

---

### 4.1 List Products

Search and browse the store catalog.

```
GET /ace/v1/products
```

**Query Parameters:**

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `q` | string | No | — | Free-text search query. |
| `category` | string | No | — | Filter by category identifier. |
| `offset` | integer | No | 0 | Pagination offset. See Section 6. |
| `limit` | integer | No | 20 | Number of results to return. Max: 100. See Section 6. |

**Response:**

**Status:** `200 OK`

```json
{
  "data": [
    {
      "id": "prod_001",
      "name": "Wireless Keyboard",
      "description": "Compact wireless keyboard with Bluetooth 5.0",
      "category": "electronics",
      "price": 49.99,
      "currency": "USD",
      "in_stock": true,
      "variants": [
        {
          "id": "var_001a",
          "name": "Black",
          "price": 49.99,
          "in_stock": true
        },
        {
          "id": "var_001b",
          "name": "White",
          "price": 49.99,
          "in_stock": false
        }
      ],
      "images": [
        "https://acme-widgets.example.com/images/kb-001.jpg"
      ],
      "metadata": {}
    }
  ],
  "total": 142,
  "offset": 0,
  "limit": 20
}
```

**Error Responses:**

| Status | Code | Description |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid API key. |
| 400 | `INVALID_REQUEST` | Invalid query parameters (e.g., negative offset). |

---

### 4.2 Get Product Detail

Retrieve full details for a single product.

```
GET /ace/v1/products/{id}
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Product identifier. |

**Response:**

**Status:** `200 OK`

```json
{
  "id": "prod_001",
  "name": "Wireless Keyboard",
  "description": "Compact wireless keyboard with Bluetooth 5.0. Features low-latency connection, rechargeable battery with 3-month life, and ergonomic key layout.",
  "category": "electronics",
  "price": 49.99,
  "currency": "USD",
  "in_stock": true,
  "variants": [
    {
      "id": "var_001a",
      "name": "Black",
      "price": 49.99,
      "in_stock": true
    },
    {
      "id": "var_001b",
      "name": "White",
      "price": 49.99,
      "in_stock": false
    }
  ],
  "images": [
    "https://acme-widgets.example.com/images/kb-001.jpg",
    "https://acme-widgets.example.com/images/kb-001-side.jpg"
  ],
  "metadata": {
    "weight_kg": 0.45,
    "dimensions_cm": "28x12x2"
  }
}
```

**Error Responses:**

| Status | Code | Description |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid API key. |
| 404 | `NOT_FOUND` | Product with the given ID does not exist. |

---

### 4.3 Quote Shipping

Get shipping options and costs for a set of items to a destination address.

```
POST /ace/v1/shipping/quote
```

**Request Body:**

```json
{
  "items": [
    {
      "product_id": "prod_001",
      "variant_id": "var_001a",
      "quantity": 2
    }
  ],
  "destination": {
    "country": "US",
    "state": "CA",
    "city": "San Francisco",
    "postal_code": "94102",
    "address_line1": "123 Main St",
    "address_line2": "Apt 4B"
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `items` | array | Yes | List of items to ship. |
| `items[].product_id` | string | Yes | Product identifier. |
| `items[].variant_id` | string | No | Variant identifier, if applicable. |
| `items[].quantity` | integer | Yes | Quantity of this item. Must be >= 1. |
| `destination` | object | Yes | Shipping destination address. |
| `destination.country` | string | Yes | ISO 3166-1 alpha-2 country code. |
| `destination.state` | string | No | State or province. |
| `destination.city` | string | No | City name. |
| `destination.postal_code` | string | Yes | Postal / ZIP code. |
| `destination.address_line1` | string | Yes | Street address. |
| `destination.address_line2` | string | No | Apartment, suite, etc. |

**Response:**

**Status:** `200 OK`

```json
{
  "shipping_options": [
    {
      "id": "ship_standard",
      "name": "Standard Shipping",
      "cost": 5.99,
      "currency": "USD",
      "estimated_days_min": 5,
      "estimated_days_max": 7
    },
    {
      "id": "ship_express",
      "name": "Express Shipping",
      "cost": 14.99,
      "currency": "USD",
      "estimated_days_min": 1,
      "estimated_days_max": 2
    }
  ]
}
```

**Error Responses:**

| Status | Code | Description |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid API key. |
| 400 | `INVALID_REQUEST` | Missing required fields or invalid address. |
| 404 | `NOT_FOUND` | One or more product IDs not found. |

---

### 4.4 Create Cart

Create a new empty cart. A cart is a temporary container for items before order placement.

```
POST /ace/v1/cart
```

**Request Body:** None (empty body or `{}`).

**Response:**

**Status:** `201 Created`

```json
{
  "id": "cart_789xyz",
  "items": [],
  "subtotal": 0.00,
  "currency": "USD",
  "created_at": "2026-03-08T14:30:00Z"
}
```

**Error Responses:**

| Status | Code | Description |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid API key. |

---

### 4.5 Add Item to Cart

Add a product to an existing cart. If the product is already in the cart, the quantity is increased.

```
POST /ace/v1/cart/{id}/items
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Cart identifier. |

**Request Body:**

```json
{
  "product_id": "prod_001",
  "variant_id": "var_001a",
  "quantity": 2
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `product_id` | string | Yes | Product identifier. |
| `variant_id` | string | No | Variant identifier. Required if the product has variants. |
| `quantity` | integer | Yes | Quantity to add. Must be >= 1. |

**Response:**

**Status:** `200 OK`

```json
{
  "id": "cart_789xyz",
  "items": [
    {
      "product_id": "prod_001",
      "variant_id": "var_001a",
      "product_name": "Wireless Keyboard",
      "variant_name": "Black",
      "quantity": 2,
      "unit_price": 49.99,
      "line_total": 99.98
    }
  ],
  "subtotal": 99.98,
  "currency": "USD",
  "created_at": "2026-03-08T14:30:00Z"
}
```

**Error Responses:**

| Status | Code | Description |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid API key. |
| 404 | `NOT_FOUND` | Cart or product not found. |
| 400 | `INVALID_REQUEST` | Missing required fields or invalid quantity. |
| 409 | `OUT_OF_STOCK` | Requested quantity exceeds available stock. |

---

### 4.6 Get Cart

Retrieve the current state of a cart, including all items and the computed subtotal.

```
GET /ace/v1/cart/{id}
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Cart identifier. |

**Response:**

**Status:** `200 OK`

```json
{
  "id": "cart_789xyz",
  "items": [
    {
      "product_id": "prod_001",
      "variant_id": "var_001a",
      "product_name": "Wireless Keyboard",
      "variant_name": "Black",
      "quantity": 2,
      "unit_price": 49.99,
      "line_total": 99.98
    }
  ],
  "subtotal": 99.98,
  "currency": "USD",
  "created_at": "2026-03-08T14:30:00Z"
}
```

**Error Responses:**

| Status | Code | Description |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid API key. |
| 404 | `NOT_FOUND` | Cart not found. |

---

### 4.7 Create Order

Convert a cart into an order. The cart is consumed and cannot be reused. The order is created with status `"pending"`.

```
POST /ace/v1/orders
```

**Request Body:**

```json
{
  "cart_id": "cart_789xyz",
  "shipping_option_id": "ship_standard",
  "shipping_address": {
    "country": "US",
    "state": "CA",
    "city": "San Francisco",
    "postal_code": "94102",
    "address_line1": "123 Main St",
    "address_line2": "Apt 4B"
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `cart_id` | string | Yes | The cart to convert into an order. |
| `shipping_option_id` | string | No | Shipping option from a previous quote. |
| `shipping_address` | object | No | Shipping destination. Same format as Section 4.3. |

**Response:**

**Status:** `201 Created`

```json
{
  "id": "order_456def",
  "status": "pending",
  "items": [
    {
      "product_id": "prod_001",
      "variant_id": "var_001a",
      "product_name": "Wireless Keyboard",
      "variant_name": "Black",
      "quantity": 2,
      "unit_price": 49.99,
      "line_total": 99.98
    }
  ],
  "subtotal": 99.98,
  "shipping_cost": 5.99,
  "total": 105.97,
  "currency": "USD",
  "shipping_option_id": "ship_standard",
  "created_at": "2026-03-08T14:35:00Z"
}
```

**Error Responses:**

| Status | Code | Description |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid API key. |
| 404 | `NOT_FOUND` | Cart not found. |
| 400 | `INVALID_REQUEST` | Cart is empty or already converted. |
| 409 | `OUT_OF_STOCK` | One or more items are no longer in stock. |

---

### 4.8 Get Order Status

Retrieve the current state of an order.

```
GET /ace/v1/orders/{id}
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Order identifier. |

**Response:**

**Status:** `200 OK`

```json
{
  "id": "order_456def",
  "status": "pending",
  "items": [
    {
      "product_id": "prod_001",
      "variant_id": "var_001a",
      "product_name": "Wireless Keyboard",
      "variant_name": "Black",
      "quantity": 2,
      "unit_price": 49.99,
      "line_total": 99.98
    }
  ],
  "subtotal": 99.98,
  "shipping_cost": 5.99,
  "total": 105.97,
  "currency": "USD",
  "shipping_option_id": "ship_standard",
  "payment_status": "unpaid",
  "created_at": "2026-03-08T14:35:00Z",
  "updated_at": "2026-03-08T14:35:00Z"
}
```

**Order Statuses:**

| Status | Description |
|---|---|
| `pending` | Order created, awaiting payment. |
| `paid` | Payment confirmed. |
| `processing` | Seller is preparing the order. |
| `shipped` | Order has been shipped. |
| `delivered` | Order has been delivered. |
| `cancelled` | Order was cancelled. |
| `refunded` | Order was refunded. |

**Error Responses:**

| Status | Code | Description |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid API key. |
| 404 | `NOT_FOUND` | Order not found. |

---

### 4.9 Initiate Payment

Initiate payment for a pending order. Returns a payment object with a URL that the buyer agent can use to complete payment (e.g., redirect a human, or invoke a payment agent).

```
POST /ace/v1/orders/{id}/pay
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Order identifier. |

**Request Body:**

```json
{
  "provider": "stripe"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `provider` | string | Yes | Payment provider. Supported values depend on the store. Common: `"stripe"`, `"mercadopago"`. |

**Response:**

**Status:** `201 Created`

```json
{
  "payment_id": "pay_abc123",
  "order_id": "order_456def",
  "provider": "stripe",
  "status": "pending",
  "payment_url": "https://checkout.stripe.com/pay/cs_live_abc123",
  "amount": 105.97,
  "currency": "USD",
  "created_at": "2026-03-08T14:36:00Z",
  "expires_at": "2026-03-08T15:36:00Z"
}
```

**Error Responses:**

| Status | Code | Description |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid API key. |
| 404 | `NOT_FOUND` | Order not found. |
| 400 | `INVALID_REQUEST` | Order is not in `pending` status, or provider is unsupported. |
| 502 | `PAYMENT_FAILED` | Payment provider returned an error. |

---

### 4.10 Get Payment Status

Check the current status of a payment for an order.

```
GET /ace/v1/orders/{id}/pay/status
```

**Path Parameters:**

| Parameter | Type | Description |
|---|---|---|
| `id` | string | Order identifier. |

**Response:**

**Status:** `200 OK`

```json
{
  "payment_id": "pay_abc123",
  "order_id": "order_456def",
  "provider": "stripe",
  "status": "completed",
  "amount": 105.97,
  "currency": "USD",
  "created_at": "2026-03-08T14:36:00Z",
  "completed_at": "2026-03-08T14:37:12Z"
}
```

**Payment Statuses:**

| Status | Description |
|---|---|
| `pending` | Payment initiated, awaiting completion. |
| `completed` | Payment successfully processed. |
| `failed` | Payment failed. |
| `expired` | Payment link expired before completion. |
| `refunded` | Payment was refunded. |

**Error Responses:**

| Status | Code | Description |
|---|---|---|
| 401 | `UNAUTHORIZED` | Missing or invalid API key. |
| 404 | `NOT_FOUND` | Order not found or no payment initiated for this order. |

---

## 5. Error Format

All error responses across the protocol use a consistent JSON format.

```json
{
  "error": "Human-readable description of what went wrong",
  "code": "MACHINE_CODE",
  "details": "Optional additional context"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `error` | string | Yes | A human-readable error message. |
| `code` | string | Yes | A machine-readable error code from the standard set. |
| `details` | string | No | Additional context (e.g., which field failed validation). |

### Standard Error Codes

| Code | HTTP Status | Description |
|---|---|---|
| `NOT_FOUND` | 404 | The requested resource does not exist. |
| `INVALID_REQUEST` | 400 | The request body or parameters are malformed or missing required fields. |
| `UNAUTHORIZED` | 401 | Authentication failed: missing or invalid API key. |
| `OUT_OF_STOCK` | 409 | The requested product or quantity is not available. |
| `PAYMENT_FAILED` | 502 | The payment provider returned an error. |
| `INTERNAL_ERROR` | 500 | An unexpected server error occurred. |

**Example error response:**

```
HTTP/1.1 404 Not Found
Content-Type: application/json

{
  "error": "Product not found",
  "code": "NOT_FOUND",
  "details": "No product exists with ID 'prod_999'"
}
```

---

## 6. Pagination

All list endpoints return paginated results using offset-based pagination.

### Query Parameters

| Parameter | Type | Default | Max | Description |
|---|---|---|---|---|
| `offset` | integer | 0 | — | Number of items to skip. |
| `limit` | integer | 20 | 100 | Number of items to return. |

### Response Wrapper

All paginated responses use this envelope:

```json
{
  "data": [ ... ],
  "total": 142,
  "offset": 0,
  "limit": 20
}
```

| Field | Type | Description |
|---|---|---|
| `data` | array | The page of results. |
| `total` | integer | Total number of matching items across all pages. |
| `offset` | integer | The offset used in this request. |
| `limit` | integer | The limit used in this request. |

**Example: fetching page 2 of products**

```
GET /ace/v1/products?offset=20&limit=20
```

```json
{
  "data": [ ... ],
  "total": 142,
  "offset": 20,
  "limit": 20
}
```

If `offset` exceeds `total`, `data` MUST be an empty array.

---

## 7. Versioning

### URL-Based Versioning

ACE uses URL path versioning. The current version prefix is `/ace/v1/`.

All endpoint paths in this specification include the version prefix. The `ace_base_url` in the discovery response includes the version (e.g., `https://example.com/ace/v1`).

### Version Policy

- **Backward-compatible changes** (adding optional fields, new endpoints) do NOT require a new version.
- **Breaking changes** (removing fields, changing semantics, altering required fields) require a new version (e.g., `/ace/v2/`).
- The `/.well-known/agent-commerce` endpoint always returns the current supported version in the `version` field.
- Stores MAY support multiple versions simultaneously by serving different `ace_base_url` values, but this is not required.

---

## 8. Seller Admin API (Informational)

ACE also defines a private Seller Admin API used by store owners to manage their store. This API is **not part of the public protocol** -- it is implementation-specific and exists behind separate authentication (typically the store owner's credentials).

The Seller Admin API covers the following functional areas:

- **Catalog Management** -- Create, read, update, and delete products. Publish and unpublish products from the buyer-facing catalog.
- **Inventory Management** -- Update stock levels, set low-stock alerts.
- **Order Management** -- View incoming orders, mark orders as fulfilled or shipped, process refunds.
- **Policies and Approvals** -- Configure store policies (return windows, shipping regions), manage buyer agent access approvals.
- **Audit Logs** -- View a history of administrative actions taken on the store.
- **API Key Management** -- Create, rotate, and revoke API keys issued to buyer agents.

Implementors are free to design the Seller Admin API as they see fit. The buyer-facing protocol (Sections 2-6) is the only interoperability surface.

---

## 9. Registry API (Informational)

The ANS Registry is a separate service that allows stores to be discovered by buyer agents. Stores register themselves with the registry after deploying an ACE-compatible server.

The registry exposes the following endpoints:

### Register a Store

```
POST /registry/v1/stores
```

**Request Body:**

```json
{
  "name": "Acme Widget Store",
  "url": "https://acme-widgets.example.com",
  "description": "High-quality widgets and accessories",
  "categories": ["electronics", "accessories"]
}
```

**Response:** `201 Created` with the registered store object including a registry-assigned `id`.

### Search Stores

```
GET /registry/v1/stores
```

**Query Parameters:** `q` (search term), `category`, `offset`, `limit`.

**Response:** `200 OK` with a paginated list of registered stores.

### Get Store Detail

```
GET /registry/v1/stores/{id}
```

**Response:** `200 OK` with full store information including its `url`, capabilities, and registration metadata.

### Health Check

```
GET /registry/v1/stores/{id}/health
```

**Response:** `200 OK` with the store's current health status, including whether its `/.well-known/agent-commerce` endpoint is reachable and returning a valid discovery response.

```json
{
  "store_id": "store_abc123",
  "status": "healthy",
  "last_checked": "2026-03-08T14:00:00Z",
  "ace_version": "ace/0.1",
  "response_time_ms": 120
}
```

The registry is an independent service and not part of the ACE store protocol itself. Its specification is maintained separately.

---

**End of ACE Protocol Specification v0.1**
