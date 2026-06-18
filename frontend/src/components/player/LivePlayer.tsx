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
import {
  createHlsConfig,
  minBufferBeforePlaySec,
} from "@/lib/player/hls-config";
import { getNetworkProfile, type NetworkProfile } from "@/lib/player/network";

interface LivePlayerProps {
  src: string;
  title?: string;
  autoPlay?: boolean;
  className?: string;
}

type QualityLevel = { height: number; label: string; index: number };

const QUALITY_STORAGE_KEY = "sahiy-quality";

function lowestBitrateLevelIndex(hls: Hls): number {
  let idx = 0;
  let min = Infinity;
  hls.levels.forEach((level, i) => {
    const br = level.bitrate ?? 0;
    if (br > 0 && br < min) {
      min = br;
      idx = i;
    }
  });
  return idx;
}

function applySlowStartAbr(hls: Hls, profile: NetworkProfile) {
  if (readSavedQuality() !== "auto") return;
  if (hls.levels.length <= 1) return;
  if (profile !== "slow" && profile !== "medium") return;

  const low = lowestBitrateLevelIndex(hls);
  hls.autoLevelCapping = low;
  hls.currentLevel = low;
}

function readSavedQuality(): "auto" | number {
  if (typeof window === "undefined") return "auto";
  const saved = localStorage.getItem(QUALITY_STORAGE_KEY);
  if (!saved || saved === "auto") return "auto";
  const height = parseInt(saved, 10);
  return Number.isFinite(height) && height > 0 ? height : "auto";
}

function persistQuality(choice: "auto" | number) {
  localStorage.setItem(
    QUALITY_STORAGE_KEY,
    choice === "auto" ? "auto" : String(choice),
  );
}

function applyQualityChoice(hls: Hls, levels: QualityLevel[]): "auto" | number {
  const saved = readSavedQuality();
  if (saved === "auto") {
    hls.currentLevel = -1;
    return "auto";
  }
  const match = levels.find((l) => l.height === saved);
  if (match) {
    hls.currentLevel = match.index;
    return match.height;
  }
  hls.currentLevel = -1;
  return "auto";
}

function bufferedAhead(video: HTMLVideoElement): number {
  const ranges = video.buffered;
  if (!ranges.length) return 0;
  return Math.max(0, ranges.end(ranges.length - 1) - video.currentTime);
}

export function LivePlayer({
  src,
  title,
  autoPlay = true,
  className,
}: LivePlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const warmupDoneRef = useRef(false);
  const hasPlayedRef = useRef(false);

  const [playing, setPlaying] = useState(false);
  const [muted, setMuted] = useState(false);
  const [volume, setVolume] = useState(0.8);
  const [fullscreen, setFullscreen] = useState(false);
  const [showControls, setShowControls] = useState(true);
  const [buffering, setBuffering] = useState(false);
  const [warming, setWarming] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [qualities, setQualities] = useState<QualityLevel[]>([]);
  const [currentQuality, setCurrentQuality] = useState<"auto" | number>("auto");
  const [showQualityMenu, setShowQualityMenu] = useState(false);
  const [latencySec, setLatencySec] = useState<number | null>(null);
  const [networkProfile] = useState<NetworkProfile>(() => getNetworkProfile());
  const [singleQuality, setSingleQuality] = useState(false);
  const hideTimer = useRef<ReturnType<typeof setTimeout>>(null);
  const latencyTimer = useRef<ReturnType<typeof setInterval>>(null);
  const warmupTimer = useRef<ReturnType<typeof setTimeout>>(null);

  const measureLatency = useCallback(() => {
    const hls = hlsRef.current;
    const video = videoRef.current;
    if (!hls || !video || !hls.liveSyncPosition) return;
    const edge = hls.liveSyncPosition;
    if (edge > 0 && video.currentTime > 0) {
      setLatencySec(Math.max(0, edge - video.currentTime));
    }
  }, []);

  const tryStartPlayback = useCallback(() => {
    const video = videoRef.current;
    const hls = hlsRef.current;
    if (!video || !hls || warmupDoneRef.current || !autoPlay) return;

    const ahead = bufferedAhead(video);
    const minBuf = minBufferBeforePlaySec(networkProfile);
    if (ahead >= minBuf) {
      warmupDoneRef.current = true;
      setWarming(false);
      if (hls.autoLevelCapping >= 0 && ahead >= minBuf + 4) {
        hls.autoLevelCapping = -1;
      }
      void video.play().catch(() => setPlaying(false));
      return;
    }

    if (warmupTimer.current) clearTimeout(warmupTimer.current);
    warmupTimer.current = setTimeout(() => {
      if (warmupDoneRef.current) return;
      warmupDoneRef.current = true;
      setWarming(false);
      void video.play().catch(() => setPlaying(false));
    }, 12_000);
  }, [autoPlay, networkProfile]);

  const initPlayer = useCallback(() => {
    const video = videoRef.current;
    if (!video || !src) return;

    setError(null);
    setWarming(true);
    setBuffering(false);
    warmupDoneRef.current = false;
    hasPlayedRef.current = false;
    if (warmupTimer.current) clearTimeout(warmupTimer.current);

    if (hlsRef.current) {
      hlsRef.current.destroy();
      hlsRef.current = null;
    }

    if (Hls.isSupported()) {
      const hls = new Hls(createHlsConfig(networkProfile));

      hlsRef.current = hls;
      hls.loadSource(src);
      hls.attachMedia(video);

      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        const levels = hls.levels
          .map((l, i) => ({
            index: i,
            height: l.height,
            label: l.height ? `${l.height}p` : `Level ${i}`,
          }))
          .filter((l) => l.height > 0)
          .sort((a, b) => b.height - a.height);

        if (levels.length <= 1) {
          hls.currentLevel = -1;
          setQualities([]);
          setCurrentQuality("auto");
          setSingleQuality(true);
        } else {
          setSingleQuality(false);
          setQualities(levels);
          const choice = applyQualityChoice(hls, levels);
          setCurrentQuality(choice === "auto" ? "auto" : choice);
          applySlowStartAbr(hls, networkProfile);
        }

        if (autoPlay) {
          tryStartPlayback();
        } else {
          setWarming(false);
        }
      });

      hls.on(Hls.Events.FRAG_BUFFERED, tryStartPlayback);
      hls.on(Hls.Events.BUFFER_APPENDED, tryStartPlayback);

      hls.on(Hls.Events.LEVEL_SWITCHED, (_, data) => {
        const level = hls.levels[data.level];
        if (hls.currentLevel === -1) {
          setCurrentQuality("auto");
        } else if (level?.height) {
          setCurrentQuality(level.height);
        }
      });

      hls.on(Hls.Events.ERROR, (_, data) => {
        if (!data.fatal) {
          if (
            data.details === Hls.ErrorDetails.BUFFER_STALLED_ERROR &&
            hls.levels.length > 1 &&
            readSavedQuality() === "auto" &&
            hls.currentLevel > 0
          ) {
            hls.nextLevel = hls.currentLevel - 1;
          }
          return;
        }
        switch (data.type) {
          case Hls.ErrorTypes.NETWORK_ERROR:
            hls.startLoad();
            break;
          case Hls.ErrorTypes.MEDIA_ERROR:
            hls.recoverMediaError();
            break;
          default:
            setError("Stream uzildi. Qayta ulanmoqda...");
            setTimeout(() => initPlayer(), 3000);
            break;
        }
      });
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = src;
      setWarming(false);
      if (autoPlay) void video.play().catch(() => setPlaying(false));
    } else {
      setError("Brauzeringiz HLS ni qo'llab-quvvatlamaydi");
      setWarming(false);
    }
  }, [src, autoPlay, tryStartPlayback, networkProfile]);

  useEffect(() => {
    initPlayer();
    return () => {
      hlsRef.current?.destroy();
      if (latencyTimer.current) clearInterval(latencyTimer.current);
      if (warmupTimer.current) clearTimeout(warmupTimer.current);
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

  function setQualityAuto() {
    if (hlsRef.current) {
      hlsRef.current.currentLevel = -1;
    }
    persistQuality("auto");
    setCurrentQuality("auto");
    setShowQualityMenu(false);
  }

  function setQualityHeight(height: number, index: number) {
    if (hlsRef.current) {
      hlsRef.current.currentLevel = index;
    }
    persistQuality(height);
    setCurrentQuality(height);
    setShowQualityMenu(false);
  }

  function jumpToLive() {
    const hls = hlsRef.current;
    const video = videoRef.current;
    if (hls?.liveSyncPosition && video) {
      video.currentTime = hls.liveSyncPosition - 2;
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
        onWaiting={() => {
          if (!hasPlayedRef.current) return;
          setBuffering(true);
          const hls = hlsRef.current;
          if (
            hls &&
            hls.levels.length > 1 &&
            readSavedQuality() === "auto" &&
            hls.currentLevel > 0
          ) {
            hls.nextLevel = hls.currentLevel - 1;
          }
        }}
        onPlaying={() => {
          hasPlayedRef.current = true;
          setBuffering(false);
          setWarming(false);
          setPlaying(true);
        }}
        onPause={() => setPlaying(false)}
      />

      {latencySec !== null && playing && !warming && (
        <button
          onClick={jumpToLive}
          className={cn(
            "absolute left-3 top-3 flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium backdrop-blur-md transition-colors",
            latencySec > 8
              ? "bg-amber-500/20 text-amber-300 hover:bg-amber-500/30"
              : "bg-black/50 text-white/90 hover:bg-black/70",
          )}
        >
          <Zap className="h-3 w-3" />
          {latencySec < 1 ? "<1s" : `${latencySec.toFixed(1)}s`}
        </button>
      )}

      {warming && !error && (
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-black/70">
          <div className="h-9 w-9 animate-pulse rounded-full bg-white/20" />
          <p className="text-sm text-white/80">Efir tayyorlanmoqda...</p>
          {singleQuality && networkProfile === "slow" && (
            <p className="max-w-xs px-4 text-center text-xs text-white/50">
              Sekin internet — OBS bitrate 2500 kbps dan past bo‘lishi tavsiya etiladi
            </p>
          )}
        </div>
      )}

      {buffering && !warming && !error && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/30 pointer-events-none">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-white/60 border-t-transparent" />
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
            {qualities.length > 1 && (
              <div className="relative">
                <button
                  onClick={() => setShowQualityMenu(!showQualityMenu)}
                  className="flex items-center gap-1 rounded-lg px-2 py-2 text-sm text-white transition-colors hover:bg-white/10"
                >
                  <Settings className="h-4 w-4" />
                  {currentQuality === "auto" ? "Auto" : `${currentQuality}p`}
                </button>
                {showQualityMenu && (
                  <div className="absolute bottom-full right-0 mb-2 min-w-[100px] overflow-hidden rounded-xl border border-white/10 bg-black/90 py-1 backdrop-blur-xl">
                    <button
                      onClick={setQualityAuto}
                      className={cn(
                        "block w-full px-4 py-2 text-left text-sm text-white hover:bg-white/10",
                        currentQuality === "auto" && "text-accent",
                      )}
                    >
                      Auto
                    </button>
                    {qualities.map((q) => (
                      <button
                        key={q.index}
                        onClick={() => setQualityHeight(q.height, q.index)}
                        className={cn(
                          "block w-full px-4 py-2 text-left text-sm text-white hover:bg-white/10",
                          currentQuality === q.height && "text-accent",
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
