import http from "node:http";

export function authenticate(req: http.IncomingMessage): { ok: boolean; actor: string; status?: number; body?: any } {
  // Accept X-ACE-Payment: mock:*
  const payment = req.headers["x-ace-payment"] as string | undefined;
  if (payment) {
    const idx = payment.indexOf(":");
    if (idx > 0) {
      const provider = payment.slice(0, idx);
      if (provider === "mock") {
        return { ok: true, actor: `payment:${payment.slice(idx + 1)}` };
      }
    }
    return { ok: false, actor: "", status: 401, body: { error: "Payment provider not supported", code: "payment_rejected" } };
  }

  // Accept X-ACE-Key (any non-empty value)
  const key = req.headers["x-ace-key"] as string | undefined;
  if (key) {
    return { ok: true, actor: `key:${key.slice(0, 8)}` };
  }

  // No auth → 402
  return {
    ok: false,
    actor: "",
    status: 402,
    body: {
      error: "Payment or API key required",
      code: "payment_required",
      pricing: {
        price: 0,
        currency: "USD",
        accepted_providers: ["mock"],
      },
    },
  };
}
