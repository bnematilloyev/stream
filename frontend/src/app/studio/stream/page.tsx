"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { getMyChannel, getIngestKey, rotateIngestKey } from "@/lib/api/channels";
import { createStream, startStream, endStream } from "@/lib/api/streams";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ShareStreamLink } from "@/components/stream/ShareStreamLink";
import { Copy, Check, RefreshCw, Radio, Square } from "lucide-react";
import Link from "next/link";
import type { Stream } from "@/types";

export default function GoLivePage() {
  const qc = useQueryClient();
  const [title, setTitle] = useState("My Live Stream");
  const [activeStream, setActiveStream] = useState<Stream | null>(null);
  const [copied, setCopied] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const channelQuery = useQuery({
    queryKey: ["my-channel"],
    queryFn: getMyChannel,
  });

  const ingestQuery = useQuery({
    queryKey: ["ingest", channelQuery.data?.slug],
    queryFn: () => getIngestKey(channelQuery.data!.slug),
    enabled: !!channelQuery.data?.slug,
  });

  const ingest = ingestQuery.data;
  const channel = channelQuery.data;

  async function copy(text: string, key: string) {
    await navigator.clipboard.writeText(text);
    setCopied(key);
    setTimeout(() => setCopied(null), 2000);
  }

  async function handleCreateAndStart() {
    if (!channel) return;
    setLoading(true);
    setError("");
    try {
      const stream = await createStream({
        channel_slug: channel.slug,
        title,
        visibility: "public",
      });
      const live = await startStream(stream.id);
      setActiveStream(live);
      await qc.invalidateQueries({ queryKey: ["my-streams"] });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Xatolik");
    } finally {
      setLoading(false);
    }
  }

  async function handleEnd() {
    if (!activeStream) return;
    setLoading(true);
    try {
      await endStream(activeStream.id);
      setActiveStream(null);
      await qc.invalidateQueries({ queryKey: ["my-streams"] });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Xatolik");
    } finally {
      setLoading(false);
    }
  }

  async function handleRotateKey() {
    if (!channel) return;
    await rotateIngestKey(channel.slug);
    await ingestQuery.refetch();
  }

  if (channelQuery.isLoading) {
    return <Skeleton className="h-96 w-full rounded-2xl" />;
  }

  if (!channel) {
    return (
      <Card>
        <CardContent className="py-12 text-center">
          <p className="text-muted">Avval kanal yarating</p>
          <Link href="/studio" className="mt-4 inline-block text-accent hover:underline">
            Studio ga qaytish
          </Link>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Efirga chiqish</h1>
        <p className="text-muted">OBS yoki boshqa dastur bilan stream qiling</p>
      </div>

      {activeStream ? (
        <Card className="border-live/30 bg-live/5">
          <CardHeader>
            <div className="flex items-center gap-2">
              <Badge variant="live">Live</Badge>
              <h2 className="font-semibold">{activeStream.title}</h2>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-sm text-muted">
              Stream ID: <code className="text-foreground">{activeStream.id}</code>
            </p>
            <Link
              href={`/live/${activeStream.id}`}
              className="text-sm text-accent hover:underline"
            >
              Tomosha sahifasini ochish →
            </Link>
            <ShareStreamLink streamId={activeStream.id} title={activeStream.title} />
            <Button variant="destructive" onClick={handleEnd} loading={loading}>
              <Square className="h-4 w-4" />
              Efirni tugatish
            </Button>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <h2 className="font-semibold">Yangi stream</h2>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="mb-1.5 block text-sm font-medium">Sarlavha</label>
              <Input value={title} onChange={(e) => setTitle(e.target.value)} />
            </div>
            <Button onClick={handleCreateAndStart} loading={loading}>
              <Radio className="h-4 w-4" />
              Stream yaratish va boshlash
            </Button>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">OBS sozlamalari</h2>
            <Button variant="ghost" size="sm" onClick={handleRotateKey}>
              <RefreshCw className="h-4 w-4" />
              Key yangilash
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {ingest ? (
            <>
              <CopyField
                label="Server (RTMP URL)"
                value={ingest.rtmp_url}
                fieldKey="rtmp"
                copied={copied}
                onCopy={copy}
              />
              <CopyField
                label="Stream Key"
                value={ingest.stream_key || ingest.key_prefix + " (mavjud key)"}
                fieldKey="key"
                copied={copied}
                onCopy={copy}
                masked={!!ingest.stream_key}
              />
              <div className="rounded-xl border border-border bg-surface-2 p-4 text-sm text-muted">
                <p className="font-medium text-foreground mb-2">Qo&apos;llanma:</p>
                <ol className="list-decimal list-inside space-y-1">
                  <li>OBS → Settings → Stream</li>
                  <li>Service: Custom</li>
                  <li>Server va Stream Key ni nusxalang</li>
                  <li>Start Streaming bosing</li>
                </ol>
              </div>
            </>
          ) : (
            <Skeleton className="h-24 w-full" />
          )}
        </CardContent>
      </Card>

      {error && (
        <p className="rounded-lg bg-red-500/10 px-4 py-3 text-sm text-red-400">{error}</p>
      )}
    </div>
  );
}

function CopyField({
  label,
  value,
  fieldKey,
  copied,
  onCopy,
  masked,
}: {
  label: string;
  value: string;
  fieldKey: string;
  copied: string | null;
  onCopy: (text: string, key: string) => void;
  masked?: boolean;
}) {
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium">{label}</label>
      <div className="flex gap-2">
        <Input
          readOnly
          value={masked ? value : value}
          className="font-mono text-xs"
        />
        <Button
          variant="secondary"
          size="icon"
          onClick={() => onCopy(value, fieldKey)}
          disabled={!value || value.includes("mavjud")}
        >
          {copied === fieldKey ? (
            <Check className="h-4 w-4 text-green-400" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
        </Button>
      </div>
    </div>
  );
}
