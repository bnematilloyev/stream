"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { listAdminChannels, updateAdminChannel } from "@/lib/api/admin";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

export default function AdminChannelsPage() {
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [marketplaceOnly, setMarketplaceOnly] = useState(false);
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-channels", page, search, marketplaceOnly],
    queryFn: () =>
      listAdminChannels({ page, limit: 20, search, marketplace_only: marketplaceOnly }),
  });

  const mutation = useMutation({
    mutationFn: ({ slug, verified }: { slug: string; verified: boolean }) =>
      updateAdminChannel(slug, verified),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-channels"] }),
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Kanallar</h1>
        <p className="text-muted">Tasdiqlash va marketplace kanallar</p>
      </div>

      <div className="flex flex-wrap gap-2">
        <Input
          placeholder="Slug yoki nom"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(1);
          }}
          className="max-w-xs"
        />
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={marketplaceOnly}
            onChange={(e) => {
              setMarketplaceOnly(e.target.checked);
              setPage(1);
            }}
          />
          Faqat marketplace
        </label>
      </div>

      <Card>
        <CardHeader>
          <h2 className="font-semibold">Kanallar ro&apos;yxati</h2>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading ? (
            <Skeleton className="h-32 w-full" />
          ) : (
            data?.data.map((ch) => (
              <div
                key={ch.id}
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border p-4"
              >
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-semibold">{ch.title}</span>
                    {ch.is_verified && <Badge>Tasdiqlangan</Badge>}
                    {ch.is_live && <Badge variant="live">LIVE</Badge>}
                  </div>
                  <p className="text-sm text-muted">@{ch.slug}</p>
                  {(ch.marketplace_seller_id ?? 0) > 0 && (
                    <p className="text-xs text-muted">
                      Seller #{ch.marketplace_seller_id} · Shop #{ch.marketplace_shop_id}
                    </p>
                  )}
                </div>
                <Button
                  size="sm"
                  variant={ch.is_verified ? "secondary" : "default"}
                  disabled={mutation.isPending}
                  onClick={() =>
                    mutation.mutate({ slug: ch.slug, verified: !ch.is_verified })
                  }
                >
                  {ch.is_verified ? "Tasdiqni olib tashlash" : "Tasdiqlash"}
                </Button>
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}
