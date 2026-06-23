"use client";

import Image from "next/image";
import Link from "next/link";
import { ShoppingBag } from "lucide-react";
import type { FeaturedProduct } from "@/lib/api/featured";

function formatPrice(price?: number, currency?: string): string | null {
  if (!price || price <= 0) return null;
  const formatted = new Intl.NumberFormat("uz-UZ").format(price);
  return currency ? `${formatted} ${currency}` : `${formatted} so'm`;
}

interface FeaturedProductCardProps {
  product: FeaturedProduct;
}

/** Efir vaqtida video ustida "hozir gaplashilayotgan mahsulot" kartasi. */
export function FeaturedProductCard({ product }: FeaturedProductCardProps) {
  const price = formatPrice(product.price, product.currency);

  const inner = (
    <div className="flex items-center gap-3 rounded-2xl border border-white/10 bg-black/70 p-2.5 pr-4 shadow-lg backdrop-blur-md transition-transform hover:scale-[1.02]">
      <div className="relative h-14 w-14 shrink-0 overflow-hidden rounded-xl bg-white/10">
        {product.image_url ? (
          <Image
            src={product.image_url}
            alt={product.title}
            fill
            sizes="56px"
            className="object-cover"
            unoptimized
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center">
            <ShoppingBag className="h-6 w-6 text-white/50" />
          </div>
        )}
      </div>
      <div className="min-w-0">
        <p className="mb-0.5 text-[10px] font-medium uppercase tracking-wide text-brand-secondary">
          Hozir efirda
        </p>
        <p className="truncate text-sm font-semibold text-white">{product.title}</p>
        <div className="flex items-center gap-2">
          {price && <p className="text-sm font-bold text-white">{price}</p>}
          {typeof product.stock === "number" && (
            <span
              className={`rounded-full px-1.5 py-0.5 text-[10px] font-semibold ${
                product.stock <= 0
                  ? "bg-red-500/80 text-white"
                  : product.stock <= 5
                    ? "bg-amber-500/90 text-black"
                    : "bg-white/15 text-white"
              }`}
            >
              {product.stock <= 0 ? "Tugadi" : `Qoldi: ${product.stock}`}
            </span>
          )}
        </div>
      </div>
    </div>
  );

  return (
    <div className="pointer-events-none absolute bottom-4 left-4 z-10 max-w-[280px]">
      {product.url ? (
        <Link
          href={product.url}
          target="_blank"
          rel="noopener noreferrer"
          className="pointer-events-auto block"
        >
          {inner}
        </Link>
      ) : (
        <div className="pointer-events-auto">{inner}</div>
      )}
    </div>
  );
}
