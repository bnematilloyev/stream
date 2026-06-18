"use client";

import dynamic from "next/dynamic";
import { Skeleton } from "@/components/ui/skeleton";
import type { Playback } from "@/types";
import type { StreamStatus } from "@/lib/user-messages";

export type PlaybackMode = "live" | "dvr" | "vod";

const LivePlayer = dynamic(
  () => import("./LivePlayer").then((m) => m.LivePlayer),
  { ssr: false, loading: () => <Skeleton className="aspect-video w-full rounded-2xl" /> },
);

const WhepPlayer = dynamic(
  () => import("./WhepPlayer").then((m) => m.WhepPlayer),
  { ssr: false, loading: () => <Skeleton className="aspect-video w-full rounded-2xl" /> },
);

export function WatchPlayer({
  playback,
  title,
  preferUltraLow = false,
  playbackMode = "live",
  streamStatus = "live",
}: {
  playback: Playback;
  title: string;
  preferUltraLow?: boolean;
  playbackMode?: PlaybackMode;
  streamStatus?: StreamStatus;
}) {
  const canWhep =
    !!playback.whep_url && playback.latency_mode === "ultra-low";
  const canHls = !!playback.url && playback.hls_ready !== false;
  const useWhep =
    canWhep &&
    (preferUltraLow || playback.playback_mode === "whep" || !canHls);

  if (useWhep) {
    return (
      <WhepPlayer
        whepUrl={playback.whep_url!}
        title={title}
        streamStatus={streamStatus}
      />
    );
  }

  if (playback.url) {
    return (
      <LivePlayer
        src={playback.url}
        title={title}
        autoPlay
        playbackMode={playbackMode}
        lowLatency={playback.playback_mode === "dual" || playback.format === "ll-hls"}
        streamStatus={streamStatus}
      />
    );
  }

  return null;
}
