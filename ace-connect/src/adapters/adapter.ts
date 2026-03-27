export interface AceProduct {
  id: string;
  name: string;
  description: string;
  category: string;
  tags: string[];
  price: { amount: number; currency: string };
  variants: AceVariant[];
  imageUrl: string;
  status: "published" | "draft";
  pricingModel: "fixed";
  pricePerRequest: 0;
  createdAt: string;
  updatedAt: string;
}

export interface AceVariant {
  id: string;
  name: string;
  sku: string;
  price: { amount: number; currency: string };
  inventory: number;
  attributes: Record<string, string>;
}

export interface EcommerceAdapter {
  name: string;
  storeName: string;
  fetchProducts(): Promise<AceProduct[]>;
}
