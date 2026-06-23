"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ShoppingBag, X } from "lucide-react";
import {
  clearFeaturedProduct,
  getFeaturedProduct,
  setFeaturedProduct,
  type FeaturedProduct,
} from "@/lib/api/featured";

interface FeaturedProductControlProps {
  streamId: string;
}

/**
 * Efir egasi uchun mahsulot ajratish paneli. Marketplace integratsiyasida bu
 * forma o'rniga haqiqiy mahsulot tanlash ro'yxati qo'yiladi — bir xil
 * setFeaturedProduct API'ni chaqiradi.
 */
export function FeaturedProductControl({ streamId }: FeaturedProductControlProps) {
  const [active, setActive] = useState<FeaturedProduct | null>(null);
  const [productId, setProductId] = useState("");
  const [title, setTitle] = useState("");
  const [price, setPrice] = useState("");
  const [imageUrl, setImageUrl] = useState("");
  const [url, setUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    getFeaturedProduct(streamId)
      .then((p) => {
        if (!cancelled) setActive(p);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [streamId]);

  async function show() {
    if (!title.trim()) {
      setError("Mahsulot nomi kerak");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const product = await setFeaturedProduct(streamId, {
        product_id: productId.trim() || title.trim(),
        title: title.trim(),
        price: price ? Number(price) : undefined,
        image_url: imageUrl.trim() || undefined,
        url: url.trim() || undefined,
      });
      setActive(product);
    } catch {
      setError("Mahsulotni ko'rsatib bo'lmadi. Qayta urinib ko'ring.");
    } finally {
      setLoading(false);
    }
  }

  async function hide() {
    setLoading(true);
    setError("");
    try {
      await clearFeaturedProduct(streamId);
      setActive(null);
    } catch {
      setError("Bekor qilib bo'lmadi.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="rounded-xl border border-border bg-surface-1 p-4">
      <div className="mb-3 flex items-center gap-2">
        <ShoppingBag className="h-4 w-4 text-brand" />
        <h3 className="font-semibold">Mahsulotni efirda ko&apos;rsatish</h3>
      </div>

      {active && (
        <div className="mb-3 flex items-center justify-between gap-3 rounded-lg border border-brand/30 bg-brand/5 p-3">
          <div className="min-w-0">
            <p className="text-xs text-brand-secondary">Hozir ko&apos;rsatilmoqda</p>
            <p className="truncate text-sm font-medium">{active.title}</p>
          </div>
          <Button variant="ghost" size="sm" onClick={hide} disabled={loading}>
            <X className="h-4 w-4" />
            Yashirish
          </Button>
        </div>
      )}

      <div className="grid gap-2 sm:grid-cols-2">
        <Input
          placeholder="Mahsulot nomi *"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <Input
          placeholder="Narxi (so'm)"
          type="number"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
        />
        <Input
          placeholder="Rasm URL"
          value={imageUrl}
          onChange={(e) => setImageUrl(e.target.value)}
        />
        <Input
          placeholder="Mahsulot havolasi"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
        />
        <Input
          placeholder="Mahsulot ID (ixtiyoriy)"
          value={productId}
          onChange={(e) => setProductId(e.target.value)}
        />
      </div>

      {error && <p className="mt-2 text-sm text-red-400">{error}</p>}

      <Button onClick={show} loading={loading} className="mt-3 w-full sm:w-auto">
        {active ? "Almashtirish" : "Ko'rsatish"}
      </Button>
    </div>
  );
}
