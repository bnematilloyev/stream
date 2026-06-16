import type Hls from "hls.js";

/** Production LL-HLS — stability-first ABR for OBS ingest and VPS origin. */
export function createHlsConfig(): Partial<Hls["config"]> {
  return {
    enableWorker: true,
    lowLatencyMode: true,

    liveSyncDurationCount: 2,
    liveMaxLatencyDurationCount: 5,
    maxLiveSyncPlaybackRate: 1.1,
    liveDurationInfinity: true,

    backBufferLength: 30,
    liveBackBufferLength: 30,
    maxBufferLength: 20,
    maxMaxBufferLength: 40,
    maxBufferSize: 48 * 1000 * 1000,
    maxBufferHole: 1.0,

    startLevel: -1,
    // Don't cap by video tag size — sidebar layout made ABR stick to 480p.
    capLevelToPlayerSize: false,
    abrEwmaDefaultEstimate: 10_000_000,
    abrBandWidthFactor: 0.85,
    abrBandWidthUpFactor: 0.9,
    abrMaxWithRealBitrate: true,
    minAutoBitrate: 2_800_000,

    fragLoadingMaxRetry: 12,
    fragLoadingRetryDelay: 700,
    manifestLoadingMaxRetry: 10,
    levelLoadingMaxRetry: 10,
    startFragPrefetch: true,
    testBandwidth: true,

    highBufferWatchdogPeriod: 1,
  };
}
