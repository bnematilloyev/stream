import { apiFetch } from "@/lib/api/client";

/** Efir vaqtida ekranda ajratib ko'rsatilayotgan mahsulot kartasi. */
export interface FeaturedProduct {
  product_id: string;
  sku_id?: string;
  title: string;
  image_url?: string;
  price?: number;
  currency?: string;
  url?: string;
  /** Qoldiq (amount_on_sale). undefined = ko'rsatilmaydi. */
  stock?: number;
}

/** Joriy ajratilgan mahsulotni oladi (efirga kech qo'shilganlar uchun). */
export async function getFeaturedProduct(
  streamId: string,
): Promise<FeaturedProduct | null> {
  const res = await apiFetch<{ data: FeaturedProduct | null }>(
    `/v1/chat/${streamId}/featured`,
  );
  return res.data;
}

/** Mahsulotni efirda ajratib ko'rsatadi (faqat efir egasi). */
export async function setFeaturedProduct(
  streamId: string,
  product: FeaturedProduct,
): Promise<FeaturedProduct> {
  const res = await apiFetch<{ data: FeaturedProduct }>(
    `/v1/chat/${streamId}/featured`,
    {
      method: "POST",
      auth: true,
      body: JSON.stringify(product),
    },
  );
  return res.data;
}

/** Ajratib ko'rsatishni bekor qiladi. */
export async function clearFeaturedProduct(streamId: string): Promise<void> {
  await apiFetch(`/v1/chat/${streamId}/featured`, {
    method: "DELETE",
    auth: true,
  });
}
