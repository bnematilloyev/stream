"use client";

import dynamic from "next/dynamic";
import { Skeleton } from "@/components/ui/skeleton";
import type { Playback } from "@/types";

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
}: {
  playback: Playback;
  title: string;
  preferUltraLow?: boolean;
}) {
  const useWhep =
    preferUltraLow &&
    playback.whep_url &&
    playback.latency_mode === "ultra-low";

  if (useWhep) {
    return <WhepPlayer whepUrl={playback.whep_url!} title={title} />;
  }

  if (playback.url) {
    return <LivePlayer src={playback.url} title={title} autoPlay />;
  }

  return null;
}
