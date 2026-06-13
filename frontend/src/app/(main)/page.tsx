"use client";

import { useQuery } from "@tanstack/react-query";
import { getLiveStreams } from "@/lib/api/streams";
import { StreamGrid, StreamGridSkeleton } from "@/components/stream/StreamGrid";
import { Radio } from "lucide-react";

export default function HomePage() {
  const { data, isLoading } = useQuery({
    queryKey: ["streams", "live"],
    queryFn: () => getLiveStreams(1, 24),
    refetchInterval: 30_000,
  });

  return (
    <div className="space-y-8">
      <section className="brand-bg relative overflow-hidden rounded-3xl p-8 text-white sm:p-12">
        <div className="relative z-10 max-w-xl">
          <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-white/20 bg-white/10 px-3 py-1 text-xs font-medium">
            <Radio className="h-3 w-3 text-gold-accent" />
            Jonli efirlar
          </div>
          <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">
            Dunyoning eng yaxshi{" "}
            <span className="text-gradient-gold">live stream</span> platformasi
          </h1>
          <p className="mt-3 leading-relaxed text-white/75">
            Past latency, ABR sifat, silliq playback. Hozir jonli efirlarni
            tomosha qiling yoki o&apos;zingiz efirga chiqing.
          </p>
        </div>
        <div className="pointer-events-none absolute -right-20 -top-20 h-64 w-64 rounded-full bg-gold-accent/20 blur-3xl" />
      </section>

      <section>
        <h2 className="mb-4 text-xl font-semibold">Hozir jonli</h2>
        {isLoading ? (
          <StreamGridSkeleton />
        ) : (
          <StreamGrid streams={data?.data ?? []} />
        )}
      </section>
    </div>
  );
}
