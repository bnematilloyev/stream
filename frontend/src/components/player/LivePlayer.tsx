"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import Hls from "hls.js";
import {
  Maximize,
  Minimize,
  Pause,
  Play,
  Volume2,
  VolumeX,
  Settings,
  Zap,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { createHlsConfig } from "@/lib/player/hls-config";

interface LivePlayerProps {
  src: string;
  title?: string;
  autoPlay?: boolean;
  className?: string;
}

type QualityLevel = { height: number; label: string; index: number };

export function LivePlayer({
  src,
  title,
  autoPlay = true,
  className,
}: LivePlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const hlsRef = useRef<Hls | null>(null);

  const [playing, setPlaying] = useState(autoPlay);
  const [muted, setMuted] = useState(false);
  const [volume, setVolume] = useState(0.8);
  const [fullscreen, setFullscreen] = useState(false);
  const [showControls, setShowControls] = useState(true);
  const [buffering, setBuffering] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [qualities, setQualities] = useState<QualityLevel[]>([]);
  const [currentQuality, setCurrentQuality] = useState(-1);
  const [showQualityMenu, setShowQualityMenu] = useState(false);
  const [latencySec, setLatencySec] = useState<number | null>(null);
  const hideTimer = useRef<ReturnType<typeof setTimeout>>(null);
  const latencyTimer = useRef<ReturnType<typeof setInterval>>(null);

  const measureLatency = useCallback(() => {
    const hls = hlsRef.current;
    const video = videoRef.current;
    if (!hls || !video || !hls.liveSyncPosition) return;
    const edge = hls.liveSyncPosition;
    if (edge > 0 && video.currentTime > 0) {
      setLatencySec(Math.max(0, edge - video.currentTime));
    }
  }, []);

  const initPlayer = useCallback(() => {
    const video = videoRef.current;
    if (!video || !src) return;

    setError(null);

    if (hlsRef.current) {
      hlsRef.current.destroy();
      hlsRef.current = null;
    }

    if (Hls.isSupported()) {
      const hls = new Hls(createHlsConfig());

      hlsRef.current = hls;
      hls.loadSource(src);
      hls.attachMedia(video);

      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        const levels = hls.levels.map((l, i) => ({
          index: i,
          height: l.height,
          label: l.height ? `${l.height}p` : `Level ${i}`,
        }));
        setQualities(levels);
        setCurrentQuality(hls.currentLevel);
        if (autoPlay) void video.play().catch(() => setPlaying(false));
      });

      hls.on(Hls.Events.LEVEL_SWITCHED, (_, data) => {
        setCurrentQuality(data.level);
      });

      hls.on(Hls.Events.ERROR, (_, data) => {
        if (data.fatal) {
          switch (data.type) {
            case Hls.ErrorTypes.NETWORK_ERROR:
              hls.startLoad();
              break;
            case Hls.ErrorTypes.MEDIA_ERROR:
              hls.recoverMediaError();
              break;
            default:
              setError("Stream uzildi. Qayta ulanmoqda...");
              setTimeout(() => initPlayer(), 2000);
              break;
          }
        }
      });
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = src;
      if (autoPlay) void video.play().catch(() => setPlaying(false));
    } else {
      setError("Brauzeringiz HLS ni qo'llab-quvvatlamaydi");
    }
  }, [src, autoPlay]);

  useEffect(() => {
    initPlayer();
    return () => {
      hlsRef.current?.destroy();
      if (latencyTimer.current) clearInterval(latencyTimer.current);
    };
  }, [initPlayer]);

  useEffect(() => {
    latencyTimer.current = setInterval(measureLatency, 1000);
    return () => {
      if (latencyTimer.current) clearInterval(latencyTimer.current);
    };
  }, [measureLatency]);

  useEffect(() => {
    const saved = localStorage.getItem("sahiy-volume");
    if (saved) setVolume(parseFloat(saved));
  }, []);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    video.volume = volume;
    video.muted = muted;
    localStorage.setItem("sahiy-volume", String(volume));
  }, [volume, muted]);

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

  function setQuality(index: number) {
    if (hlsRef.current) {
      hlsRef.current.currentLevel = index;
      setCurrentQuality(index);
    }
    setShowQualityMenu(false);
  }

  function jumpToLive() {
    const hls = hlsRef.current;
    const video = videoRef.current;
    if (hls?.liveSyncPosition && video) {
      video.currentTime = hls.liveSyncPosition - 0.5;
    }
  }

  function resetHideTimer() {
    setShowControls(true);
    if (hideTimer.current) clearTimeout(hideTimer.current);
    hideTimer.current = setTimeout(() => {
      if (playing) setShowControls(false);
    }, 3000);
  }

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.target instanceof HTMLInputElement) return;
      switch (e.key) {
        case " ":
        case "k":
          e.preventDefault();
          togglePlay();
          break;
        case "f":
          toggleFullscreen();
          break;
        case "m":
          setMuted((m) => !m);
          break;
        case "l":
          jumpToLive();
          break;
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  return (
    <div
      ref={containerRef}
      className={cn(
        "group relative aspect-video overflow-hidden rounded-2xl bg-black",
        className,
      )}
      onMouseMove={resetHideTimer}
      onMouseLeave={() => playing && setShowControls(false)}
    >
      <video
        ref={videoRef}
        className="h-full w-full"
        playsInline
        onClick={togglePlay}
        onWaiting={() => setBuffering(true)}
        onPlaying={() => {
          setBuffering(false);
          setPlaying(true);
        }}
        onPause={() => setPlaying(false)}
      />

      {latencySec !== null && playing && (
        <button
          onClick={jumpToLive}
          className={cn(
            "absolute left-3 top-3 flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium backdrop-blur-md transition-colors",
            latencySec > 5
              ? "bg-amber-500/20 text-amber-300 hover:bg-amber-500/30"
              : "bg-black/50 text-white/90 hover:bg-black/70",
          )}
        >
          <Zap className="h-3 w-3" />
          {latencySec < 1 ? "<1s" : `${latencySec.toFixed(1)}s`}
        </button>
      )}

      {buffering && !error && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/40">
          <div className="h-10 w-10 animate-spin rounded-full border-2 border-white border-t-transparent" />
        </div>
      )}

      {error && (
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-black/80 p-6 text-center">
          <p className="text-sm text-white/80">{error}</p>
          <button
            onClick={initPlayer}
            className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white"
          >
            Qayta urinish
          </button>
        </div>
      )}

      <div
        className={cn(
          "absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/90 via-black/50 to-transparent p-4 transition-opacity duration-300",
          showControls ? "opacity-100" : "opacity-0",
        )}
      >
        <div className="mb-3 h-1 overflow-hidden rounded-full bg-white/20">
          <div className="h-full w-full rounded-full bg-live animate-pulse" />
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={togglePlay}
            className="rounded-lg p-2 text-white transition-colors hover:bg-white/10"
          >
            {playing ? <Pause className="h-5 w-5" /> : <Play className="h-5 w-5" />}
          </button>

          <button
            onClick={() => setMuted(!muted)}
            className="rounded-lg p-2 text-white transition-colors hover:bg-white/10"
          >
            {muted || volume === 0 ? (
              <VolumeX className="h-5 w-5" />
            ) : (
              <Volume2 className="h-5 w-5" />
            )}
          </button>

          <input
            type="range"
            min={0}
            max={1}
            step={0.05}
            value={muted ? 0 : volume}
            onChange={(e) => {
              setVolume(parseFloat(e.target.value));
              setMuted(false);
            }}
            className="hidden w-20 accent-accent sm:block"
          />

          {title && (
            <span className="ml-2 hidden truncate text-sm text-white/80 sm:block">
              {title}
            </span>
          )}

          <div className="ml-auto flex items-center gap-1">
            {qualities.length > 0 && (
              <div className="relative">
                <button
                  onClick={() => setShowQualityMenu(!showQualityMenu)}
                  className="flex items-center gap-1 rounded-lg px-2 py-2 text-sm text-white transition-colors hover:bg-white/10"
                >
                  <Settings className="h-4 w-4" />
                  {currentQuality === -1
                    ? "Auto"
                    : qualities.find((q) => q.index === currentQuality)?.label}
                </button>
                {showQualityMenu && (
                  <div className="absolute bottom-full right-0 mb-2 min-w-[100px] overflow-hidden rounded-xl border border-white/10 bg-black/90 py-1 backdrop-blur-xl">
                    <button
                      onClick={() => setQuality(-1)}
                      className={cn(
                        "block w-full px-4 py-2 text-left text-sm text-white hover:bg-white/10",
                        currentQuality === -1 && "text-accent",
                      )}
                    >
                      Auto
                    </button>
                    {qualities.map((q) => (
                      <button
                        key={q.index}
                        onClick={() => setQuality(q.index)}
                        className={cn(
                          "block w-full px-4 py-2 text-left text-sm text-white hover:bg-white/10",
                          currentQuality === q.index && "text-accent",
                        )}
                      >
                        {q.label}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}

            <button
              onClick={toggleFullscreen}
              className="rounded-lg p-2 text-white transition-colors hover:bg-white/10"
            >
              {fullscreen ? (
                <Minimize className="h-5 w-5" />
              ) : (
                <Maximize className="h-5 w-5" />
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
