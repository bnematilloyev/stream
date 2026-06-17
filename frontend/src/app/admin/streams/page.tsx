"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import {
  forceEndStream,
  listAdminLiveStreams,
} from "@/lib/api/admin";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

export default function AdminStreamsPage() {
  const qc = useQueryClient();
  const [marketplaceOnly, setMarketplaceOnly] = useState(false);
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-live-streams", page, marketplaceOnly],
    queryFn: () =>
      listAdminLiveStreams({ page, limit: 20, marketplace_only: marketplaceOnly }),
    refetchInterval: 15000,
  });

  const endMutation = useMutation({
    mutationFn: forceEndStream,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["admin-live-streams"] }),
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">Jonli efirlar</h1>
          <p className="text-muted">Real-time monitoring va force-end</p>
        </div>
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
          <h2 className="font-semibold">Live ro&apos;yxat</h2>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading ? (
            <Skeleton className="h-32 w-full" />
          ) : data?.data.length === 0 ? (
            <p className="text-muted">Hozir jonli efir yo&apos;q</p>
          ) : (
            data?.data.map((st) => (
              <div
                key={st.id}
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border p-4"
              >
                <div>
                  <div className="flex items-center gap-2">
                    <Badge variant="live">LIVE</Badge>
                    <span className="font-semibold">{st.title}</span>
                  </div>
                  <p className="mt-1 text-sm text-muted">
                    @{st.channel_slug} · {st.viewer_count} tomoshabin
                  </p>
                  {(st.marketplace_seller_id ?? 0) > 0 && (
                    <p className="text-xs text-muted">
                      Seller #{st.marketplace_seller_id} · Shop #{st.marketplace_shop_id}
                    </p>
                  )}
                </div>
                <div className="flex gap-2">
                  <Link href={`/live/${st.id}`}>
                    <Button size="sm" variant="secondary">
                      Ko&apos;rish
                    </Button>
                  </Link>
                  <Button
                    size="sm"
                    disabled={endMutation.isPending}
                    onClick={() => endMutation.mutate(st.id)}
                  >
                    To&apos;xtatish
                  </Button>
                </div>
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}
