import type Hls from "hls.js";

/** Production LL-HLS — stability-first ABR for OBS ingest and VPS origin. */
export function createHlsConfig(): Partial<Hls["config"]> {
  return {
    enableWorker: true,
    lowLatencyMode: true,

    liveSyncDurationCount: 3,
    liveMaxLatencyDurationCount: 8,
    maxLiveSyncPlaybackRate: 1.1,
    liveDurationInfinity: true,

    backBufferLength: 30,
    liveBackBufferLength: 30,
    maxBufferLength: 20,
    maxMaxBufferLength: 40,
    maxBufferSize: 48 * 1000 * 1000,
    maxBufferHole: 1.0,

    startLevel: -1,
    capLevelToPlayerSize: true,
    abrEwmaDefaultEstimate: 4_000_000,
    abrBandWidthFactor: 0.8,
    abrBandWidthUpFactor: 0.7,
    abrMaxWithRealBitrate: true,
    minAutoBitrate: 600_000,

    fragLoadingMaxRetry: 12,
    fragLoadingRetryDelay: 700,
    manifestLoadingMaxRetry: 10,
    levelLoadingMaxRetry: 10,
    startFragPrefetch: true,
    testBandwidth: true,

    highBufferWatchdogPeriod: 1,
  };
}
