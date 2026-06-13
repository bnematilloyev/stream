"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { WHIPClient } from "@eyevinn/whip-web-client";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Camera,
  CameraOff,
  Mic,
  MicOff,
  Radio,
  Square,
  SwitchCamera,
} from "lucide-react";
import Link from "next/link";
import { endStream, startStream } from "@/lib/api/streams";
import { ShareStreamLink } from "@/components/stream/ShareStreamLink";
import { cameraBlockedReason, canUseCamera } from "@/lib/media";
import { broadcastPageUrl, whipEndpoint } from "@/lib/whip";

interface CameraBroadcastProps {
  streamId: string;
  streamKey: string;
  title: string;
  whipBaseUrl?: string;
  onEnd: () => void;
}

export function CameraBroadcast({
  streamId,
  title,
  whipBaseUrl,
  onEnd,
}: CameraBroadcastProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const whipRef = useRef<WHIPClient | null>(null);
  const streamRef = useRef<MediaStream | null>(null);

  const [live, setLive] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [facingMode, setFacingMode] = useState<"user" | "environment">("user");
  const [videoEnabled, setVideoEnabled] = useState(true);
  const [audioEnabled, setAudioEnabled] = useState(true);

  const getMedia = useCallback(async (facing: "user" | "environment") => {
    const blocked = cameraBlockedReason();
    if (blocked) {
      throw new Error(blocked);
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((t) => t.stop());
    }
    const media = await navigator.mediaDevices!.getUserMedia({
      video: {
        facingMode: facing,
        width: { ideal: 1280 },
        height: { ideal: 720 },
        frameRate: { ideal: 30 },
      },
      audio: {
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true,
      },
    });
    streamRef.current = media;
    if (videoRef.current) {
      videoRef.current.srcObject = media;
    }
    return media;
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await getMedia(facingMode);
      } catch (e) {
        if (!cancelled) {
          const msg =
            e instanceof Error ? e.message : "Kameraga ruxsat berilmadi";
          setError(
            msg.includes("HTTPS") || msg.includes("qo'llab-quvvatlamaydi")
              ? msg
              : "Kameraga ruxsat berilmadi. Brauzer sozlamalaridan kamera/mikrofonni yoqing.",
          );
        }
      }
    })();
    return () => {
      cancelled = true;
      streamRef.current?.getTracks().forEach((t) => t.stop());
      void whipRef.current?.destroy();
    };
  }, [facingMode, getMedia]);

  async function startBroadcast() {
    setLoading(true);
    setError("");
    try {
      const media = streamRef.current ?? (await getMedia(facingMode));
      const endpoint = whipEndpoint(streamId, whipBaseUrl);

      const client = new WHIPClient({
        endpoint,
        opts: {
          debug: process.env.NODE_ENV === "development",
          iceServers: [
            { urls: "stun:stun.l.google.com:19302" },
            { urls: "stun:stun1.l.google.com:19302" },
          ],
          iceGatheringTimeout: 3000,
        },
      });

      await client.ingest(media);
      whipRef.current = client;
      await startStream(streamId);
      setLive(true);
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Efir boshlanmadi";
      setError(
        msg.includes("fetch") || msg.includes("Failed")
          ? `WHIP serverga ulanib bo'lmadi (${whipEndpoint(streamId, whipBaseUrl)}). MediaMTX ishlayotganini tekshiring: docker compose -f infra/docker/docker-compose.yml up -d mediamtx`
          : msg,
      );
    } finally {
      setLoading(false);
    }
  }

  async function stopBroadcast() {
    setLoading(true);
    setError("");
    try {
      await whipRef.current?.destroy();
      whipRef.current = null;
      streamRef.current?.getTracks().forEach((t) => t.stop());
      setLive(false);
      try {
        await endStream(streamId);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Efirni tugatib bo'lmadi");
        return;
      }
      onEnd();
    } finally {
      setLoading(false);
    }
  }

  function toggleVideo() {
    const track = streamRef.current?.getVideoTracks()[0];
    if (track) {
      track.enabled = !track.enabled;
      setVideoEnabled(track.enabled);
    }
  }

  function toggleAudio() {
    const track = streamRef.current?.getAudioTracks()[0];
    if (track) {
      track.enabled = !track.enabled;
      setAudioEnabled(track.enabled);
    }
  }

  async function flipCamera() {
    if (live) return;
    setFacingMode((f) => (f === "user" ? "environment" : "user"));
  }

  const blocked = cameraBlockedReason();

  return (
    <div className="space-y-4">
      {blocked && (
        <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-900 dark:text-amber-200">
          <p className="font-medium">Kamera ishlamaydi</p>
          <p className="mt-1">{blocked}</p>
          <a
            href="https://stream.shopla.uz/studio/broadcast"
            className="mt-2 inline-block font-medium text-brand underline"
          >
            https://stream.shopla.uz/studio/broadcast
          </a>
        </div>
      )}
      <div className="relative aspect-video overflow-hidden rounded-2xl bg-black">
        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted
          className="h-full w-full object-cover mirror"
        />
        {live && (
          <div className="absolute left-4 top-4">
            <Badge variant="live">LIVE</Badge>
          </div>
        )}
        <div className="absolute bottom-4 left-1/2 flex -translate-x-1/2 gap-2">
          <button
            onClick={toggleAudio}
            className="rounded-full bg-black/60 p-3 text-white backdrop-blur-sm"
          >
            {audioEnabled ? <Mic className="h-5 w-5" /> : <MicOff className="h-5 w-5" />}
          </button>
          <button
            onClick={toggleVideo}
            className="rounded-full bg-black/60 p-3 text-white backdrop-blur-sm"
          >
            {videoEnabled ? (
              <Camera className="h-5 w-5" />
            ) : (
              <CameraOff className="h-5 w-5" />
            )}
          </button>
          {!live && (
            <button
              onClick={flipCamera}
              className="rounded-full bg-black/60 p-3 text-white backdrop-blur-sm"
            >
              <SwitchCamera className="h-5 w-5" />
            </button>
          )}
        </div>
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
        {!live ? (
          <Button
            onClick={startBroadcast}
            loading={loading}
            size="lg"
            className="w-full sm:w-auto"
            disabled={!canUseCamera()}
          >
            <Radio className="h-5 w-5" />
            Efirni boshlash
          </Button>
        ) : (
          <Button
            variant="destructive"
            onClick={stopBroadcast}
            loading={loading}
            size="lg"
            className="w-full sm:w-auto"
          >
            <Square className="h-5 w-5" />
            Efirni tugatish
          </Button>
        )}
        {live && (
          <Link href={`/live/${streamId}`} className="w-full sm:w-auto">
            <Button variant="secondary" className="w-full">
              Tomosha sahifasi
            </Button>
          </Link>
        )}
      </div>

      <p className="text-sm text-muted">
        <strong>{title}</strong> — kamera orqali jonli efir (WebRTC/WHIP)
      </p>

      {error && (
        <p className="rounded-xl bg-red-500/10 px-4 py-3 text-sm text-red-400">
          {error}
        </p>
      )}

      {live && <ShareStreamLink streamId={streamId} title={title} />}

      <div className="rounded-xl border border-border bg-surface-2 p-4 text-xs text-muted">
        <p className="font-medium text-foreground mb-1">Telefonda ishlatish:</p>
        <ul className="list-disc list-inside space-y-1">
          <li>Kompyuter va telefon bir Wi-Fi da bo&apos;lsin</li>
          <li>
            Telefonda: <code className="break-all">{broadcastPageUrl()}</code>
          </li>
          <li>
            WHIP:{" "}
            <code className="break-all">
              {whipEndpoint(streamId, whipBaseUrl)}
            </code>
          </li>
        </ul>
      </div>

      <style jsx>{`
        .mirror {
          transform: scaleX(-1);
        }
      `}</style>
    </div>
  );
}
