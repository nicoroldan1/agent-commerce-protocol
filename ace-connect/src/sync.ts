import { EcommerceAdapter } from "./adapters/adapter.js";
import { InMemoryStore } from "./server/store.js";

export class SyncManager {
  private adapter: EcommerceAdapter;
  private store: InMemoryStore;
  private intervalMs: number;
  private timer: NodeJS.Timeout | null = null;
  private onSync?: (count: number) => void;

  constructor(adapter: EcommerceAdapter, store: InMemoryStore, intervalSec: number, onSync?: (count: number) => void) {
    this.adapter = adapter;
    this.store = store;
    this.intervalMs = intervalSec * 1000;
    this.onSync = onSync;
  }

  async syncOnce(): Promise<number> {
    const products = await this.adapter.fetchProducts();
    this.store.replaceProducts(products);
    this.onSync?.(products.length);
    return products.length;
  }

  start() {
    this.timer = setInterval(() => {
      this.syncOnce().catch((err) =>
        console.error(`[sync] Error: ${err.message}`)
      );
    }, this.intervalMs);
  }

  stop() {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }
}
