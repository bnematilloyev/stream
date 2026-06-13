"use client";

import { useQuery } from "@tanstack/react-query";
import { getLiveStreams } from "@/lib/api/streams";
import { StreamGrid, StreamGridSkeleton } from "@/components/stream/StreamGrid";

export default function BrowsePage() {
  const { data, isLoading } = useQuery({
    queryKey: ["streams", "live", "browse"],
    queryFn: () => getLiveStreams(1, 48),
    refetchInterval: 15_000,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Jonli streamlar</h1>
        <p className="text-muted">Barcha faol efirlar</p>
      </div>
      {isLoading ? (
        <StreamGridSkeleton />
      ) : (
        <StreamGrid streams={data?.data ?? []} />
      )}
    </div>
  );
}
