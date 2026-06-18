"use client";

import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { getChannel } from "@/lib/api/channels";
import { getChannelStreams } from "@/lib/api/streams";
import { ChannelHeader } from "@/components/channel/ChannelHeader";
import { StreamGrid, StreamGridSkeleton } from "@/components/stream/StreamGrid";
import { MainLayout } from "@/components/layout/MainLayout";
import { Skeleton } from "@/components/ui/skeleton";

export default function ChannelPage() {
  const { slug } = useParams<{ slug: string }>();

  const channelQuery = useQuery({
    queryKey: ["channel", slug],
    queryFn: () => getChannel(slug),
    enabled: !!slug,
  });

  const streamsQuery = useQuery({
    queryKey: ["channel-streams", slug],
    queryFn: () => getChannelStreams(slug),
    enabled: !!slug,
  });

  if (channelQuery.isLoading) {
    return (
      <MainLayout>
        <Skeleton className="h-48 w-full rounded-2xl" />
      </MainLayout>
    );
  }

  if (!channelQuery.data) {
    return (
      <MainLayout>
        <p className="text-center text-muted py-20">Kanal topilmadi</p>
      </MainLayout>
    );
  }

  return (
    <MainLayout>
      <div className="space-y-8">
        <ChannelHeader channel={channelQuery.data} />
        <section>
          <h2 className="mb-4 text-xl font-semibold">Streamlar</h2>
          {streamsQuery.isLoading ? (
            <StreamGridSkeleton />
          ) : (
            <>
              {(() => {
                const all = streamsQuery.data?.data ?? [];
                const live = all.filter((s) => s.status === "live");
                const replays = all.filter((s) => s.status === "ended");
                const other = all.filter(
                  (s) => s.status !== "live" && s.status !== "ended",
                );
                return (
                  <div className="space-y-8">
                    {live.length > 0 && (
                      <div>
                        <h3 className="mb-3 text-sm font-medium text-muted">Jonli</h3>
                        <StreamGrid streams={live} />
                      </div>
                    )}
                    {replays.length > 0 && (
                      <div>
                        <h3 className="mb-3 text-sm font-medium text-muted">
                          Yozuvlar
                        </h3>
                        <StreamGrid streams={replays} />
                      </div>
                    )}
                    {other.length > 0 && <StreamGrid streams={other} />}
                    {all.length === 0 && (
                      <p className="text-sm text-muted">Hali stream yo&apos;q</p>
                    )}
                  </div>
                );
              })()}
            </>
          )}
        </section>
      </div>
    </MainLayout>
  );
}
