import crypto from "node:crypto";
import { AceProduct } from "../adapters/adapter.js";

interface CartItem {
  product_id: string;
  variant_id: string;
  quantity: number;
  price: { amount: number; currency: string };
}

interface Cart {
  id: string;
  items: CartItem[];
  total: { amount: number; currency: string };
  created_at: string;
  updated_at: string;
}

interface OrderItem {
  product_id: string;
  product_name: string;
  variant_id: string;
  quantity: number;
  price: { amount: number; currency: string };
}

interface Order {
  id: string;
  cart_id: string;
  items: OrderItem[];
  total: { amount: number; currency: string };
  status: string;
  payment: Payment | null;
  created_at: string;
  updated_at: string;
}

interface Payment {
  id: string;
  order_id: string;
  status: string;
  provider: string;
  amount: { amount: number; currency: string };
  external_id: string;
  payment_url: string;
  created_at: string;
}

function genId(prefix: string): string {
  return `${prefix}_${crypto.randomBytes(8).toString("hex")}`;
}

export class InMemoryStore {
  products = new Map<string, AceProduct>();
  private carts = new Map<string, Cart>();
  private orders = new Map<string, Order>();
  private currency = "USD";

  setCurrency(c: string) {
    this.currency = c;
  }

  replaceProducts(products: AceProduct[]) {
    const newMap = new Map<string, AceProduct>();
    for (const p of products) newMap.set(p.id, p);
    this.products = newMap;
  }

  listProducts(query?: string, category?: string, offset = 0, limit = 20) {
    let arr = Array.from(this.products.values()).filter(
      (p) => p.status === "published"
    );
    if (category) arr = arr.filter((p) => p.category.toLowerCase() === category.toLowerCase());
    if (query) {
      const q = query.toLowerCase();
      arr = arr.filter(
        (p) => p.name.toLowerCase().includes(q) || p.description.toLowerCase().includes(q)
      );
    }
    const total = arr.length;
    return { data: arr.slice(offset, offset + limit), total };
  }

  getProduct(id: string): AceProduct | undefined {
    const p = this.products.get(id);
    return p && p.status === "published" ? p : undefined;
  }

  createCart(): Cart {
    const cart: Cart = {
      id: genId("cart"),
      items: [],
      total: { amount: 0, currency: this.currency },
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    this.carts.set(cart.id, cart);
    return cart;
  }

  getCart(id: string): Cart | undefined {
    return this.carts.get(id);
  }

  addCartItem(cartId: string, productId: string, variantId: string | undefined, quantity: number): Cart | { error: string; code: string } {
    const cart = this.carts.get(cartId);
    if (!cart) return { error: "Cart not found", code: "cart_not_found" };

    const product = this.getProduct(productId);
    if (!product) return { error: "Product not found", code: "product_not_found" };

    let price = product.price;
    if (variantId) {
      const variant = product.variants.find((v) => v.id === variantId);
      if (!variant) return { error: "Variant not found", code: "variant_not_found" };
      price = variant.price;
    } else if (product.variants.length > 0) {
      price = product.variants[0].price;
    }

    cart.items.push({ product_id: productId, variant_id: variantId || "", quantity, price });
    cart.total.amount = cart.items.reduce((sum, i) => sum + i.price.amount * i.quantity, 0);
    cart.updated_at = new Date().toISOString();
    return cart;
  }

  createOrder(cartId: string): Order | { error: string; code: string } {
    const cart = this.carts.get(cartId);
    if (!cart) return { error: "Cart not found", code: "cart_not_found" };
    if (cart.items.length === 0) return { error: "Cart is empty", code: "empty_cart" };

    const items: OrderItem[] = cart.items.map((i) => ({
      product_id: i.product_id,
      product_name: this.products.get(i.product_id)?.name || "Unknown",
      variant_id: i.variant_id,
      quantity: i.quantity,
      price: i.price,
    }));

    const order: Order = {
      id: genId("ord"),
      cart_id: cartId,
      items,
      total: { ...cart.total },
      status: "pending",
      payment: null,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    this.orders.set(order.id, order);
    return order;
  }

  getOrder(id: string): Order | undefined {
    return this.orders.get(id);
  }

  payOrder(orderId: string, provider: string): Payment | { error: string; code: string } {
    const order = this.orders.get(orderId);
    if (!order) return { error: "Order not found", code: "order_not_found" };
    if (order.status !== "pending") return { error: "Order not pending", code: "invalid_status" };

    const payment: Payment = {
      id: genId("pay"),
      order_id: orderId,
      status: "completed",
      provider,
      amount: order.total,
      external_id: `mock_${orderId}`,
      payment_url: `https://pay.example.com/mock/${orderId}`,
      created_at: new Date().toISOString(),
    };

    order.payment = payment;
    order.status = "paid";
    order.updated_at = new Date().toISOString();
    return payment;
  }

  getPaymentByOrderId(orderId: string): Payment | undefined {
    return this.orders.get(orderId)?.payment || undefined;
  }
}
