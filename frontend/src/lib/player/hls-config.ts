import type Hls from "hls.js";

/** Barqaror jonli efir — biroz kechikish, kam uzilish (mijoz UX uchun). */
export function createHlsConfig(): Partial<Hls["config"]> {
  return {
    enableWorker: true,
    lowLatencyMode: false,

    // Live edgedan 4 segment orqada (~8s @ 2s segment) — buffer to‘lib boshlaydi.
    liveSyncDurationCount: 4,
    liveMaxLatencyDurationCount: 12,
    maxLiveSyncPlaybackRate: 1.0,
    liveDurationInfinity: true,

    backBufferLength: 45,
    liveBackBufferLength: 0,
    maxBufferLength: 50,
    maxMaxBufferLength: 90,
    maxBufferSize: 64 * 1000 * 1000,
    maxBufferHole: 0.5,

    startLevel: -1,
    capLevelToPlayerSize: false,
    abrEwmaDefaultEstimate: 5_000_000,
    abrBandWidthFactor: 0.8,
    abrBandWidthUpFactor: 0.7,
    abrMaxWithRealBitrate: true,
    minAutoBitrate: 1_400_000,

    fragLoadingMaxRetry: 24,
    fragLoadingRetryDelay: 1000,
    manifestLoadingMaxRetry: 40,
    manifestLoadingRetryDelay: 1500,
    levelLoadingMaxRetry: 16,
    levelLoadingRetryDelay: 1000,
    startFragPrefetch: true,
    testBandwidth: true,

    nudgeMaxRetry: 8,
    highBufferWatchdogPeriod: 2,
  };
}

/** Ultra-low latency (WHEP bo‘lmaganda ixtiyoriy). */
export function createLowLatencyHlsConfig(): Partial<Hls["config"]> {
  return {
    ...createHlsConfig(),
    lowLatencyMode: true,
    liveSyncDurationCount: 3,
    liveMaxLatencyDurationCount: 6,
    maxLiveSyncPlaybackRate: 1.05,
    maxBufferLength: 25,
    maxMaxBufferLength: 40,
  };
}
