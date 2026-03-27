import http from "node:http";
import { URL } from "node:url";
import { InMemoryStore } from "./store.js";
import { authenticate } from "./auth.js";

export interface ServerConfig {
  port: number;
  storeName: string;
  storeId: string;
  baseUrl: string;
  currency: string;
}

function writeJSON(res: http.ServerResponse, status: number, body: any) {
  res.setHeader("Content-Type", "application/json");
  res.setHeader("X-ACE-Price", "0.00");
  res.setHeader("X-ACE-Currency", "USD");
  res.writeHead(status);
  res.end(JSON.stringify(body));
}

function writeError(res: http.ServerResponse, status: number, code: string, message: string) {
  res.setHeader("Content-Type", "application/json");
  res.writeHead(status);
  res.end(JSON.stringify({ error: message, code }));
}

function readBody(req: http.IncomingMessage): Promise<any> {
  return new Promise((resolve, reject) => {
    let data = "";
    req.on("data", (chunk) => (data += chunk));
    req.on("end", () => {
      try { resolve(data ? JSON.parse(data) : {}); }
      catch { reject(new Error("Invalid JSON")); }
    });
    req.on("error", reject);
  });
}

export function createServer(store: InMemoryStore, config: ServerConfig): http.Server {
  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url || "/", `http://localhost:${config.port}`);
    const path = url.pathname;
    const method = req.method || "GET";

    try {
      // Public endpoints
      if (method === "GET" && path === "/.well-known/agent-commerce") {
        return writeJSON(res, 200, {
          store_id: config.storeId,
          name: config.storeName,
          version: "1.0.0",
          ace_base_url: `${config.baseUrl}/ace/v1`,
          capabilities: ["catalog", "cart", "orders", "payments", "shipping"],
          auth: { type: "api_key", header: "X-ACE-Key" },
          payment_auth: {
            enabled: true,
            header: "X-ACE-Payment",
            providers: ["mock"],
            default_currency: config.currency,
          },
          currencies: [config.currency],
        });
      }

      if (method === "GET" && path === "/ace/v1/pricing") {
        return writeJSON(res, 200, {
          default_currency: config.currency,
          endpoints: [
            { method: "GET", path: "/ace/v1/products", price: 0 },
            { method: "GET", path: "/ace/v1/products/{id}", price: 0 },
            { method: "POST", path: "/ace/v1/cart", price: 0 },
            { method: "POST", path: "/ace/v1/cart/{id}/items", price: 0 },
            { method: "POST", path: "/ace/v1/orders", price: 0 },
            { method: "POST", path: "/ace/v1/orders/{id}/pay", price: 0 },
          ],
        });
      }

      // Auth-protected endpoints
      if (path.startsWith("/ace/v1/")) {
        const auth = authenticate(req);
        if (!auth.ok) {
          return writeJSON(res, auth.status!, auth.body);
        }
      }

      // Routes
      if (method === "GET" && path === "/ace/v1/products") {
        const q = url.searchParams.get("q") || undefined;
        const cat = url.searchParams.get("category") || undefined;
        const offset = parseInt(url.searchParams.get("offset") || "0");
        const limit = Math.min(parseInt(url.searchParams.get("limit") || "20"), 100);
        const result = store.listProducts(q, cat, offset, limit);
        return writeJSON(res, 200, { ...result, offset, limit });
      }

      const productMatch = path.match(/^\/ace\/v1\/products\/(.+)$/);
      if (method === "GET" && productMatch) {
        const product = store.getProduct(productMatch[1]);
        if (!product) return writeError(res, 404, "not_found", "Product not found");
        return writeJSON(res, 200, product);
      }

      if (method === "POST" && path === "/ace/v1/cart") {
        return writeJSON(res, 201, store.createCart());
      }

      const cartItemMatch = path.match(/^\/ace\/v1\/cart\/(.+)\/items$/);
      if (method === "POST" && cartItemMatch) {
        const body = await readBody(req);
        const result = store.addCartItem(cartItemMatch[1], body.product_id, body.variant_id, body.quantity || 1);
        if ("error" in result) return writeError(res, 400, result.code, result.error);
        return writeJSON(res, 200, result);
      }

      const cartMatch = path.match(/^\/ace\/v1\/cart\/(.+)$/);
      if (method === "GET" && cartMatch && !cartMatch[1].includes("/")) {
        const cart = store.getCart(cartMatch[1]);
        if (!cart) return writeError(res, 404, "not_found", "Cart not found");
        return writeJSON(res, 200, cart);
      }

      if (method === "POST" && path === "/ace/v1/orders") {
        const body = await readBody(req);
        const result = store.createOrder(body.cart_id);
        if ("error" in result) return writeError(res, 400, result.code, result.error);
        return writeJSON(res, 201, result);
      }

      const payMatch = path.match(/^\/ace\/v1\/orders\/(.+)\/pay$/);
      if (method === "POST" && payMatch) {
        const body = await readBody(req);
        const result = store.payOrder(payMatch[1], body.provider || "mock");
        if ("error" in result) return writeError(res, 400, result.code, result.error);
        return writeJSON(res, 201, result);
      }

      const payStatusMatch = path.match(/^\/ace\/v1\/orders\/(.+)\/pay\/status$/);
      if (method === "GET" && payStatusMatch) {
        const payment = store.getPaymentByOrderId(payStatusMatch[1]);
        if (!payment) return writeError(res, 404, "not_found", "Payment not found");
        return writeJSON(res, 200, payment);
      }

      const orderMatch = path.match(/^\/ace\/v1\/orders\/(.+)$/);
      if (method === "GET" && orderMatch && !orderMatch[1].includes("/")) {
        const order = store.getOrder(orderMatch[1]);
        if (!order) return writeError(res, 404, "not_found", "Order not found");
        return writeJSON(res, 200, order);
      }

      if (method === "POST" && path === "/ace/v1/shipping/quote") {
        return writeJSON(res, 200, {
          options: [
            { id: "ship_standard", name: "Standard Shipping", price: { amount: 599, currency: config.currency }, estimated_days: 7 },
            { id: "ship_express", name: "Express Shipping", price: { amount: 1299, currency: config.currency }, estimated_days: 3 },
            { id: "ship_overnight", name: "Overnight Shipping", price: { amount: 2499, currency: config.currency }, estimated_days: 1 },
          ],
        });
      }

      writeError(res, 404, "not_found", "Endpoint not found");
    } catch (err: any) {
      writeError(res, 500, "internal_error", err.message || "Internal server error");
    }
  });

  return server;
}
