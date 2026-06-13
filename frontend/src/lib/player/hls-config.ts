import type Hls from "hls.js";

/** Production LL-HLS tuning — 2–4s glass-to-glass target. */
export function createHlsConfig(): Partial<Hls["config"]> {
  return {
    enableWorker: true,
    lowLatencyMode: true,

    // Aggressive live edge tracking
    liveSyncDurationCount: 2,
    liveMaxLatencyDurationCount: 5,
    maxLiveSyncPlaybackRate: 1.15,
    liveDurationInfinity: true,

    // Minimal buffering — reduces perceived latency
    backBufferLength: 0,
    liveBackBufferLength: 0,
    maxBufferLength: 8,
    maxMaxBufferLength: 16,
    maxBufferSize: 20 * 1000 * 1000,
    maxBufferHole: 0.5,

    // Fast ABR for quality under load
    startLevel: -1,
    capLevelToPlayerSize: true,
    abrEwmaDefaultEstimate: 8_000_000,
    abrBandWidthFactor: 0.9,
    abrBandWidthUpFactor: 0.7,
    abrMaxWithRealBitrate: true,

    // Resilience
    fragLoadingMaxRetry: 8,
    fragLoadingRetryDelay: 500,
    manifestLoadingMaxRetry: 6,
    levelLoadingMaxRetry: 6,
    startFragPrefetch: true,
    testBandwidth: true,

    // LL-HLS part handling
    highBufferWatchdogPeriod: 1,
  };
}
