"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { getMyChannel } from "@/lib/api/channels";
import { endStream, getChannelStreams } from "@/lib/api/streams";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Camera, Radio, Users, Video } from "lucide-react";
import { CreateChannelForm } from "@/components/studio/CreateChannelForm";

export default function StudioDashboard() {
  const qc = useQueryClient();
  const [endingId, setEndingId] = useState<string | null>(null);

  const channelQuery = useQuery({
    queryKey: ["my-channel"],
    queryFn: getMyChannel,
    retry: false,
  });

  const streamsQuery = useQuery({
    queryKey: ["my-streams", channelQuery.data?.slug],
    queryFn: () => getChannelStreams(channelQuery.data!.slug, 1, 10),
    enabled: !!channelQuery.data?.slug,
  });

  const channel = channelQuery.data;
  const liveCount =
    streamsQuery.data?.data.filter((s) => s.status === "live").length ?? 0;

  async function handleEndStream(streamId: string) {
    setEndingId(streamId);
    try {
      await endStream(streamId);
      await qc.invalidateQueries({ queryKey: ["my-streams"] });
    } finally {
      setEndingId(null);
    }
  }

  if (channelQuery.isLoading) {
    return <Skeleton className="h-64 w-full rounded-2xl" />;
  }

  if (channelQuery.isError) {
    return (
      <Card>
        <CardContent className="py-12 text-center">
          <p className="text-red-400">
            Kanal yuklanmadi:{" "}
            {channelQuery.error instanceof Error
              ? channelQuery.error.message
              : "server xatosi"}
          </p>
          <p className="mt-2 text-sm text-muted">
            Migration yoki user-service tekshiring (500 — odatda DB migration
            qilinmagan).
          </p>
        </CardContent>
      </Card>
    );
  }

  if (!channel) {
    return (
      <Card>
        <CardHeader>
          <h1 className="text-2xl font-bold">Kanal yarating</h1>
          <p className="text-muted">Efirga chiqish uchun avval kanal kerak</p>
        </CardHeader>
        <CardContent>
          <CreateChannelForm />
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">Creator Studio</h1>
          <p className="text-muted">@{channel.slug}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Link href="/studio/broadcast">
            <Button>
              <Camera className="h-4 w-4" />
              Kamera efir
            </Button>
          </Link>
          <Link href="/studio/stream">
            <Button variant="secondary">
              <Radio className="h-4 w-4" />
              OBS / RTMP
            </Button>
          </Link>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="flex items-center gap-4 pt-6">
            <div className="rounded-xl bg-brand-light p-3">
              <Users className="h-5 w-5 text-brand" />
            </div>
            <div>
              <p className="text-2xl font-bold">{channel.follower_count}</p>
              <p className="text-sm text-muted">Obunachilar</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 pt-6">
            <div className="rounded-xl bg-live/15 p-3">
              <Radio className="h-5 w-5 text-live" />
            </div>
            <div>
              <p className="text-2xl font-bold">{liveCount}</p>
              <p className="text-sm text-muted">Jonli streamlar</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="flex items-center gap-4 pt-6">
            <div className="rounded-xl bg-surface-3 p-3">
              <Video className="h-5 w-5 text-muted" />
            </div>
            <div>
              <p className="text-2xl font-bold">
                {streamsQuery.data?.pagination.total ?? 0}
              </p>
              <p className="text-sm text-muted">Jami streamlar</p>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <h2 className="font-semibold">So&apos;nggi streamlar</h2>
        </CardHeader>
        <CardContent>
          {streamsQuery.data?.data.length === 0 ? (
            <p className="text-sm text-muted">Hali stream yo&apos;q</p>
          ) : (
            <ul className="space-y-3">
              {streamsQuery.data?.data.slice(0, 5).map((s) => (
                <li
                  key={s.id}
                  className="flex items-center justify-between rounded-xl border border-border px-4 py-3"
                >
                  <div>
                    <p className="font-medium">{s.title}</p>
                    <p className="text-xs text-muted">
                      {new Date(s.created_at_unix * 1000).toLocaleDateString("uz-UZ")}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    {s.status === "live" && (
                      <Button
                        variant="destructive"
                        size="sm"
                        loading={endingId === s.id}
                        onClick={() => handleEndStream(s.id)}
                      >
                        Tugatish
                      </Button>
                    )}
                    <Badge variant={s.status === "live" ? "live" : "outline"}>
                      {s.status}
                    </Badge>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
