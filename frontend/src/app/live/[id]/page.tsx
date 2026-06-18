"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { getStreamPlayback, getStream, recordViewerHeartbeat } from "@/lib/api/streams";
import { getChannel } from "@/lib/api/channels";
import { WatchPlayer, type PlaybackMode } from "@/components/player/WatchPlayer";
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

  const stream = streamQuery.data;
  const isLive = stream?.status === "live";
  const isReplay = stream?.status === "ended";
  const playerMode: PlaybackMode = isLive ? "dvr" : "vod";

  const playbackQuery = useQuery({
    queryKey: ["playback", id, stream?.status],
    queryFn: ({ signal }) =>
      getStreamPlayback(id, signal, { warmup: isLive }),
    enabled: !!id && (isLive || isReplay),
    staleTime: isReplay ? 300_000 : 30_000,
    retry: isReplay ? 1 : 2,
    refetchInterval: (query) => {
      if (!isLive) return false;
      const pb = query.state.data;
      if (pb?.hls_ready) return false;
      return 3000;
    },
  });

  const channelQuery = useQuery({
    queryKey: ["channel", streamQuery.data?.channel_slug],
    queryFn: () => getChannel(streamQuery.data!.channel_slug),
    enabled: !!streamQuery.data?.channel_slug,
  });

  const playback = playbackQuery.data;
  const isUltraLow = playback?.latency_mode === "ultra-low";
  const hlsReady = playback?.hls_ready !== false;
  const [ultraLow, setUltraLow] = useState(true);
  const playbackReady = !!(playback?.whep_url || (playback?.url && hlsReady));

  useEffect(() => {
    if (!id || !isLive || !playbackReady) return;

    const key = "sahiy-viewer-session";
    let sessionID = localStorage.getItem(key);
    if (!sessionID) {
      sessionID =
        typeof crypto !== "undefined" && "randomUUID" in crypto
          ? crypto.randomUUID()
          : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
      localStorage.setItem(key, sessionID);
    }

    const send = () => {
      if (document.visibilityState === "hidden") return;
      void recordViewerHeartbeat(id, sessionID).catch(() => undefined);
    };

    send();
    const timer = setInterval(send, 25_000);
    return () => clearInterval(timer);
  }, [id, isLive, playbackReady]);

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
                {isLive && playback.playback_mode === "dual" && playback.url && (
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant={ultraLow ? "default" : "secondary"}
                      onClick={() => setUltraLow(true)}
                    >
                      Ultra-low (&lt;2s)
                    </Button>
                    <Button
                      size="sm"
                      variant={!ultraLow ? "default" : "secondary"}
                      onClick={() => setUltraLow(false)}
                      disabled={!hlsReady}
                      title={!hlsReady ? "HLS hali tayyorlanmoqda" : undefined}
                    >
                      LL-HLS (CDN scale)
                    </Button>
                  </div>
                )}
                <WatchPlayer
                  playback={playback}
                  title={stream?.title ?? ""}
                  preferUltraLow={isUltraLow ? ultraLow : false}
                  playbackMode={playerMode}
                />
              </div>
            ) : isLive ? (
              <div className="flex aspect-video flex-col items-center justify-center gap-3 rounded-2xl border border-border bg-surface-1">
                <div className="h-9 w-9 animate-pulse rounded-full bg-brand/30" />
                <p className="text-muted">Efir tayyorlanmoqda...</p>
              </div>
            ) : isReplay ? (
              <div className="flex aspect-video items-center justify-center rounded-2xl border border-border bg-surface-1">
                <p className="text-muted">Yozuv topilmadi yoki hali tayyor emas</p>
              </div>
            ) : (
              <div className="flex aspect-video items-center justify-center rounded-2xl border border-border bg-surface-1">
                <p className="text-muted">Bu stream hozir jonli emas</p>
              </div>
            )}

            {stream && (
              <div>
                <div className="flex flex-wrap items-center gap-2">
                  <h1 className="text-xl font-bold sm:text-2xl">{stream.title}</h1>
                  {isLive && playbackReady && (
                      <Badge variant="live">Live</Badge>
                    )}
                  {isReplay && playbackReady && (
                      <Badge variant="outline">Yozuv</Badge>
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
              live={isLive && playbackReady}
            />
          </aside>
        </div>
      </div>
    </div>
  );
}
