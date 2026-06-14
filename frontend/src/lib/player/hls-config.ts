import type Hls from "hls.js";

/** Production LL-HLS — quality-first ABR for VPS origin (few viewers, high bitrate ladder). */
export function createHlsConfig(): Partial<Hls["config"]> {
  return {
    enableWorker: true,
    lowLatencyMode: true,

    liveSyncDurationCount: 2,
    liveMaxLatencyDurationCount: 6,
    maxLiveSyncPlaybackRate: 1.1,
    liveDurationInfinity: true,

    backBufferLength: 0,
    liveBackBufferLength: 0,
    maxBufferLength: 12,
    maxMaxBufferLength: 24,
    maxBufferSize: 32 * 1000 * 1000,
    maxBufferHole: 0.5,

    // Prefer higher renditions when bandwidth allows
    startLevel: -1,
    capLevelToPlayerSize: false,
    abrEwmaDefaultEstimate: 12_000_000,
    abrBandWidthFactor: 0.95,
    abrBandWidthUpFactor: 0.85,
    abrMaxWithRealBitrate: true,
    minAutoBitrate: 2_500_000,

    fragLoadingMaxRetry: 8,
    fragLoadingRetryDelay: 500,
    manifestLoadingMaxRetry: 6,
    levelLoadingMaxRetry: 6,
    startFragPrefetch: true,
    testBandwidth: true,

    highBufferWatchdogPeriod: 1,
  };
}
