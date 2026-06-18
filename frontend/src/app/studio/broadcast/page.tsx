"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getMyChannel, getIngestKey } from "@/lib/api/channels";
import { createStream, endStream } from "@/lib/api/streams";
import { formatUserError } from "@/lib/user-messages";
import { CameraBroadcast } from "@/components/broadcast/CameraBroadcast";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import Link from "next/link";
import type { Stream } from "@/types";

export default function BroadcastPage() {
  const [title, setTitle] = useState("Kamera efir");
  const [activeStream, setActiveStream] = useState<Stream | null>(null);
  const [whipBaseUrl, setWhipBaseUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const channelQuery = useQuery({
    queryKey: ["my-channel"],
    queryFn: getMyChannel,
    retry: false,
  });

  async function prepareStream() {
    const channel = channelQuery.data;
    if (!channel) return;
    setLoading(true);
    setError("");
    try {
      const ingest = await getIngestKey(channel.slug);
      if (!ingest.whip_base_url) {
        throw new Error("WHIP server sozlanmagan");
      }
      const stream = await createStream({
        channel_slug: channel.slug,
        title,
        visibility: "public",
        latency_mode: "ultra-low",
        ingest_protocol: "whip",
      });
      setWhipBaseUrl(ingest.whip_base_url);
      setActiveStream(stream);
    } catch (e) {
      setError(formatUserError(e));
    } finally {
      setLoading(false);
    }
  }

  async function handleEnd() {
    if (!activeStream) return;
    try {
      await endStream(activeStream.id);
    } finally {
      setActiveStream(null);
      setWhipBaseUrl("");
    }
  }

  if (channelQuery.isLoading) {
    return <Skeleton className="h-96 w-full rounded-2xl" />;
  }

  if (!channelQuery.data) {
    return (
      <Card>
        <CardContent className="py-12 text-center">
          <p className="text-muted">Avval kanal yarating</p>
          <Link href="/studio" className="mt-4 inline-block text-accent hover:underline">
            Studio →
          </Link>
        </CardContent>
      </Card>
    );
  }

  if (!activeStream || !whipBaseUrl) {
    return (
      <div className="mx-auto w-full max-w-lg space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Kamera bilan efir</h1>
          <p className="text-muted">
            Telefon yoki kompyuter kamerasidan to&apos;g&apos;ridan-to&apos;g&apos;ri efirga chiqing — OBS kerak emas
          </p>
        </div>
        <Card>
          <CardHeader>
            <h2 className="font-semibold">Stream sozlamalari</h2>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="mb-1.5 block text-sm font-medium">Sarlavha</label>
              <Input value={title} onChange={(e) => setTitle(e.target.value)} />
            </div>
            <Button onClick={prepareStream} loading={loading} className="w-full" size="lg">
              Kamerani ochish
            </Button>
            {error && (
              <p className="text-sm text-red-400">{error}</p>
            )}
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold">Kamera bilan efir</h1>
        <p className="text-muted">@{channelQuery.data.slug}</p>
      </div>
      <CameraBroadcast
        streamId={activeStream.id}
        title={activeStream.title}
        whipBaseUrl={whipBaseUrl}
        onEnd={handleEnd}
      />
    </div>
  );
}
