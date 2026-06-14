"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { getPlayback, getStream } from "@/lib/api/streams";
import { getChannel } from "@/lib/api/channels";
import { WatchPlayer } from "@/components/player/WatchPlayer";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Header } from "@/components/layout/Header";
import { formatViewerCount, timeAgo } from "@/lib/utils";
import { ChatPanel } from "@/components/chat/ChatPanel";
import { Eye } from "lucide-react";

export default function WatchPage() {
  const { id } = useParams<{ id: string }>();

  const streamQuery = useQuery({
    queryKey: ["stream", id],
    queryFn: () => getStream(id),
    enabled: !!id,
  });

  const playbackQuery = useQuery({
    queryKey: ["playback", id],
    queryFn: () => getPlayback(id),
    enabled: !!id && streamQuery.data?.status === "live",
    refetchInterval: 60_000,
    retry: 3,
  });

  const channelQuery = useQuery({
    queryKey: ["channel", streamQuery.data?.channel_slug],
    queryFn: () => getChannel(streamQuery.data!.channel_slug),
    enabled: !!streamQuery.data?.channel_slug,
  });

  const [ultraLow, setUltraLow] = useState(false);
  const stream = streamQuery.data;
  const playback = playbackQuery.data;

  return (
    <div className="min-h-screen bg-background">
      <Header />
      <div className="mx-auto max-w-[1400px] px-4 py-6 lg:px-6">
        <div className="grid gap-6 lg:grid-cols-[1fr_340px]">
          <div className="space-y-4">
            {playbackQuery.isLoading || streamQuery.isLoading ? (
              <Skeleton className="aspect-video w-full rounded-2xl" />
            ) : playback?.url || playback?.whep_url ? (
              <div className="space-y-3">
                {playback.playback_mode === "dual" && (
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant={ultraLow ? "secondary" : "default"}
                      onClick={() => setUltraLow(true)}
                    >
                      Ultra-low (&lt;2s)
                    </Button>
                    <Button
                      size="sm"
                      variant={!ultraLow ? "secondary" : "default"}
                      onClick={() => setUltraLow(false)}
                    >
                      LL-HLS (CDN scale)
                    </Button>
                  </div>
                )}
                <WatchPlayer
                  playback={playback}
                  title={stream?.title ?? ""}
                  preferUltraLow={ultraLow || playback.latency_mode === "ultra-low"}
                />
              </div>
            ) : (
              <div className="flex aspect-video items-center justify-center rounded-2xl border border-border bg-surface-1">
                <p className="text-muted">
                  {stream?.status === "live"
                    ? "Stream yuklanmoqda..."
                    : "Bu stream hozir jonli emas"}
                </p>
              </div>
            )}

            {stream && (
              <div>
                <div className="flex flex-wrap items-center gap-2">
                  <h1 className="text-xl font-bold sm:text-2xl">{stream.title}</h1>
                  {stream.status === "live" &&
                    (playback?.url || playback?.whep_url) && (
                      <Badge variant="live">Live</Badge>
                    )}
                </div>
                <div className="mt-2 flex flex-wrap items-center gap-4 text-sm text-muted">
                  <Link
                    href={`/channel/${stream.channel_slug}`}
                    className="font-medium text-foreground hover:text-brand-secondary"
                  >
                    {stream.channel_title}
                  </Link>
                  {stream.viewer_count > 0 && (
                    <span className="flex items-center gap-1">
                      <Eye className="h-4 w-4" />
                      {formatViewerCount(stream.viewer_count)} tomoshabin
                    </span>
                  )}
                  {stream.started_at_unix > 0 && (
                    <span>{timeAgo(stream.started_at_unix)} boshlangan</span>
                  )}
                </div>
                {stream.description && (
                  <p className="mt-3 text-sm text-muted leading-relaxed">
                    {stream.description}
                  </p>
                )}
              </div>
            )}
          </div>

          <aside className="space-y-4">
            {channelQuery.data && (
              <div className="rounded-2xl border border-border bg-surface-1 p-4">
                <Link
                  href={`/channel/${channelQuery.data.slug}`}
                  className="flex items-center gap-3"
                >
                  <div className="accent-gradient flex h-12 w-12 items-center justify-center rounded-xl font-bold text-brand">
                    {channelQuery.data.title.charAt(0)}
                  </div>
                  <div>
                    <p className="font-semibold">{channelQuery.data.title}</p>
                    <p className="text-sm text-muted">
                      {formatViewerCount(channelQuery.data.follower_count)} obunachi
                    </p>
                  </div>
                </Link>
              </div>
            )}

            <ChatPanel
              streamId={id}
              live={stream?.status === "live" && !!(playback?.url || playback?.whep_url)}
            />
          </aside>
        </div>
      </div>
    </div>
  );
}
