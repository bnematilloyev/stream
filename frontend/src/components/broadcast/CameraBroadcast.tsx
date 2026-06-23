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
import { FeaturedProductControl } from "@/components/broadcast/FeaturedProductControl";
import { cameraBlockedReason, canUseCamera } from "@/lib/media";
import { broadcastPageUrl, whipEndpoint } from "@/lib/whip";
import { whipBroadcastMessage } from "@/lib/user-messages";

const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_BASE_DELAY_MS = 2000;
const RECONNECT_MAX_DELAY_MS = 30000;

interface CameraBroadcastProps {
  streamId: string;
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
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isLiveRef = useRef(false);

  const [live, setLive] = useState(false);
  const [reconnecting, setReconnecting] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [facingMode, setFacingMode] = useState<"user" | "environment">("user");
  const [videoEnabled, setVideoEnabled] = useState(true);
  const [audioEnabled, setAudioEnabled] = useState(true);

  function clearReconnectTimer() {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
  }

  const getMedia = useCallback(async (facing: "user" | "environment") => {
    const blocked = cameraBlockedReason();
    if (blocked) throw new Error(blocked);
    streamRef.current?.getTracks().forEach((t) => t.stop());
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
    if (videoRef.current) videoRef.current.srcObject = media;
    return media;
  }, []);

  function buildWhipClient(endpoint: string): WHIPClient {
    return new WHIPClient({
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
  }

  function setupConnectionMonitor(client: WHIPClient, endpoint: string) {
    const pc = (client as unknown as { pc?: RTCPeerConnection }).pc;
    if (!pc) return;
    pc.addEventListener("connectionstatechange", () => {
      if (!isLiveRef.current) return;
      if (pc.connectionState === "failed" || pc.connectionState === "disconnected") {
        scheduleReconnect(endpoint);
      }
    });
  }

  function scheduleReconnect(endpoint: string) {
    if (reconnectAttemptsRef.current >= MAX_RECONNECT_ATTEMPTS) {
      setLive(false);
      isLiveRef.current = false;
      setReconnecting(false);
      setError("Ulanish uzildi. Sahifani yangilab, qayta urinib ko'ring.");
      return;
    }

    const delay = Math.min(
      RECONNECT_BASE_DELAY_MS * 2 ** reconnectAttemptsRef.current,
      RECONNECT_MAX_DELAY_MS,
    );
    reconnectAttemptsRef.current += 1;
    setReconnecting(true);
    setError("");

    reconnectTimerRef.current = setTimeout(async () => {
      if (!isLiveRef.current || !streamRef.current) return;
      try {
        await whipRef.current?.destroy();
        const client = buildWhipClient(endpoint);
        await client.ingest(streamRef.current);
        whipRef.current = client;
        reconnectAttemptsRef.current = 0;
        setReconnecting(false);
        setupConnectionMonitor(client, endpoint);
      } catch {
        scheduleReconnect(endpoint);
      }
    }, delay);
  }

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await getMedia(facingMode);
      } catch (e) {
        if (!cancelled) {
          const msg = e instanceof Error ? e.message : "Kameraga ruxsat berilmadi";
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

      const client = buildWhipClient(endpoint);
      await client.ingest(media);

      try {
        await startStream(streamId);
      } catch (e) {
        await client.destroy();
        throw e;
      }

      whipRef.current = client;
      reconnectAttemptsRef.current = 0;
      isLiveRef.current = true;
      setLive(true);
      setupConnectionMonitor(client, endpoint);
    } catch (e) {
      setError(whipBroadcastMessage(e));
    } finally {
      setLoading(false);
    }
  }

  async function stopBroadcast() {
    setLoading(true);
    setError("");
    clearReconnectTimer();
    isLiveRef.current = false;
    try {
      await whipRef.current?.destroy();
      whipRef.current = null;
      streamRef.current?.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
      setLive(false);
      setReconnecting(false);
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

  function flipCamera() {
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
            href={broadcastPageUrl()}
            className="mt-2 inline-block font-medium text-brand underline"
          >
            {broadcastPageUrl()}
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
          <div className="absolute left-4 top-4 flex items-center gap-2">
            <Badge variant="live">LIVE</Badge>
            {reconnecting && (
              <span className="rounded-full bg-yellow-500/80 px-2 py-0.5 text-xs font-medium text-black">
                Qayta ulanmoqda…
              </span>
            )}
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
          <Link href={`/live/${streamId}`} target="_blank" rel="noopener noreferrer" className="w-full sm:w-auto">
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

      {live && <FeaturedProductControl streamId={streamId} />}

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
