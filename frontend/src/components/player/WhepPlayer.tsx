"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Maximize, Minimize, Pause, Play, Volume2, VolumeX, Zap } from "lucide-react";
import { cn } from "@/lib/utils";
import { PlayerMessage } from "@/components/player/PlayerMessage";
import {
  connectionLostMessage,
  type StreamStatus,
  whepPlaybackMessage,
} from "@/lib/user-messages";

interface WhepPlayerProps {
  whepUrl: string;
  title?: string;
  className?: string;
  streamStatus?: StreamStatus;
}

/** Ultra-low latency WebRTC playback (~0.5–2s). */
export function WhepPlayer({
  whepUrl,
  title,
  className,
  streamStatus = "live",
}: WhepPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const resourceRef = useRef<string | null>(null);

  const [playing, setPlaying] = useState(true);
  const [muted, setMuted] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [connected, setConnected] = useState(false);
  const [canRetry, setCanRetry] = useState(true);

  const connect = useCallback(async () => {
    setError(null);
    const video = videoRef.current;
    if (!video) return;

    if (streamStatus === "ended") {
      setCanRetry(false);
      setError(whepPlaybackMessage(404, "ended"));
      return;
    }

    try {
      await pcRef.current?.close();
      const pc = new RTCPeerConnection({
        iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
        bundlePolicy: "max-bundle",
      });
      pcRef.current = pc;

      pc.ontrack = (ev) => {
        if (ev.streams[0]) {
          video.srcObject = ev.streams[0];
          void video.play();
        }
      };

      pc.onconnectionstatechange = () => {
        if (pc.connectionState === "connected") setConnected(true);
        if (pc.connectionState === "failed") {
          setCanRetry(true);
          setError(connectionLostMessage());
        }
      };

      pc.addTransceiver("video", { direction: "recvonly" });
      pc.addTransceiver("audio", { direction: "recvonly" });

      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);

      const res = await fetch(whepUrl, {
        method: "POST",
        headers: { "Content-Type": "application/sdp" },
        body: offer.sdp,
      });

      if (!res.ok) {
        setCanRetry(streamStatus === "live" && res.status !== 404);
        setError(whepPlaybackMessage(res.status, streamStatus));
        return;
      }

      const answer = await res.text();
      const location = res.headers.get("Location");
      if (location) resourceRef.current = location;

      await pc.setRemoteDescription({ type: "answer", sdp: answer });
      setConnected(true);
      setCanRetry(true);
    } catch {
      setCanRetry(streamStatus === "live");
      setError(whepPlaybackMessage(null, streamStatus));
    }
  }, [whepUrl, streamStatus]);

  useEffect(() => {
    void connect();
    return () => {
      void pcRef.current?.close();
      if (resourceRef.current) {
        void fetch(resourceRef.current, { method: "DELETE" }).catch(() => {});
      }
    };
  }, [connect]);

  function togglePlay() {
    const video = videoRef.current;
    if (!video) return;
    if (video.paused) {
      void video.play();
      setPlaying(true);
    } else {
      video.pause();
      setPlaying(false);
    }
  }

  function toggleFullscreen() {
    const el = containerRef.current;
    if (!el) return;
    if (!document.fullscreenElement) {
      void el.requestFullscreen();
      setFullscreen(true);
    } else {
      void document.exitFullscreen();
      setFullscreen(false);
    }
  }

  return (
    <div
      ref={containerRef}
      className={cn("relative aspect-video overflow-hidden rounded-2xl bg-black", className)}
    >
      <video
        ref={videoRef}
        className="h-full w-full"
        playsInline
        autoPlay
        onClick={togglePlay}
      />

      {connected && !error && (
        <div className="absolute left-3 top-3 flex items-center gap-1 rounded-lg bg-emerald-500/20 px-2 py-1 text-xs font-medium text-emerald-300 backdrop-blur-md">
          <Zap className="h-3 w-3" />
          Ultra-low (&lt;2s)
        </div>
      )}

      {error && (
        <PlayerMessage
          message={error}
          onRetry={canRetry ? () => void connect() : undefined}
        />
      )}

      <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/90 to-transparent p-4">
        <div className="flex items-center gap-3">
          <button onClick={togglePlay} className="rounded-lg p-2 text-white hover:bg-white/10">
            {playing ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
          </button>
          <button
            onClick={() => setMuted(!muted)}
            className="rounded-lg p-2 text-white hover:bg-white/10"
          >
            {muted ? <VolumeX className="h-5 w-5" /> : <Volume2 className="h-5 w-5" />}
          </button>
          {title && <span className="truncate text-sm text-white/80">{title}</span>}
          <button
            onClick={toggleFullscreen}
            className="ml-auto rounded-lg p-2 text-white hover:bg-white/10"
          >
            {fullscreen ? <Minimize className="h-5 w-5" /> : <Maximize className="h-5 w-5" />}
          </button>
        </div>
      </div>
    </div>
  );
}
