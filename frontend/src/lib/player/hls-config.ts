import type Hls from "hls.js";
import type { NetworkProfile } from "./network";
import { getNetworkProfile, isMobileViewport } from "./network";

/** YouTube/Twitch uslubi: tarmoq tezligiga qarab buffer va ABR aggressivligi. */
export function createHlsConfig(
  profile: NetworkProfile = getNetworkProfile(),
): Partial<Hls["config"]> {
  const mobile = isMobileViewport();

  const base: Partial<Hls["config"]> = {
    enableWorker: true,
    lowLatencyMode: false,
    liveDurationInfinity: true,
    maxLiveSyncPlaybackRate: 1.0,

    startLevel: -1,
    capLevelToPlayerSize: mobile,
    abrMaxWithRealBitrate: true,
    startFragPrefetch: true,
    testBandwidth: true,

    fragLoadingMaxRetry: 24,
    fragLoadingRetryDelay: 1000,
    manifestLoadingMaxRetry: 40,
    manifestLoadingRetryDelay: 1500,
    levelLoadingMaxRetry: 16,
    levelLoadingRetryDelay: 1000,
    nudgeMaxRetry: 8,
    highBufferWatchdogPeriod: 2,
  };

  switch (profile) {
    case "slow":
      return {
        ...base,
        // Sekin mobil / 3G: katta buffer, pastdan boshlash, sekin yuqoriga chiqish.
        liveSyncDurationCount: 6,
        liveMaxLatencyDurationCount: 14,
        backBufferLength: 30,
        liveBackBufferLength: 0,
        maxBufferLength: 70,
        maxMaxBufferLength: 120,
        maxBufferSize: 80 * 1000 * 1000,
        maxBufferHole: 0.35,
        abrEwmaDefaultEstimate: 900_000,
        abrBandWidthFactor: 0.65,
        abrBandWidthUpFactor: 0.45,
        minAutoBitrate: 400_000,
      };
    case "medium":
      return {
        ...base,
        liveSyncDurationCount: 5,
        liveMaxLatencyDurationCount: 12,
        backBufferLength: 40,
        liveBackBufferLength: 0,
        maxBufferLength: 55,
        maxMaxBufferLength: 90,
        maxBufferSize: 64 * 1000 * 1000,
        maxBufferHole: 0.45,
        abrEwmaDefaultEstimate: 2_500_000,
        abrBandWidthFactor: 0.72,
        abrBandWidthUpFactor: 0.55,
        minAutoBitrate: 700_000,
      };
    case "fast":
      return {
        ...base,
        liveSyncDurationCount: 3,
        liveMaxLatencyDurationCount: 8,
        backBufferLength: 35,
        liveBackBufferLength: 0,
        maxBufferLength: 40,
        maxMaxBufferLength: 60,
        maxBufferSize: 48 * 1000 * 1000,
        maxBufferHole: 0.5,
        abrEwmaDefaultEstimate: 8_000_000,
        abrBandWidthFactor: 0.85,
        abrBandWidthUpFactor: 0.8,
        minAutoBitrate: 1_400_000,
      };
    default:
      return {
        ...base,
        liveSyncDurationCount: 4,
        liveMaxLatencyDurationCount: 12,
        backBufferLength: 45,
        liveBackBufferLength: 0,
        maxBufferLength: 50,
        maxMaxBufferLength: 90,
        maxBufferSize: 64 * 1000 * 1000,
        maxBufferHole: 0.5,
        abrEwmaDefaultEstimate: 2_000_000,
        abrBandWidthFactor: 0.75,
        abrBandWidthUpFactor: 0.6,
        minAutoBitrate: 700_000,
      };
  }
}

export function minBufferBeforePlaySec(profile: NetworkProfile): number {
  switch (profile) {
    case "slow":
      return 6;
    case "medium":
      return 5;
    case "fast":
      return 3;
    default:
      return 4;
  }
}

/**
 * Jonli efir + DVR (pause, orqaga/oldinga surish).
 * liveMaxLatencyDurationCount cheklanmasa, pause yoki seek dan keyin HLS.live edge ga sakraydi.
 */
export function createDvrHlsConfig(
  profile: NetworkProfile = getNetworkProfile(),
): Partial<Hls["config"]> {
  return {
    ...createHlsConfig(profile),
    // Faqat *Count variant — Duration bilan aralashtirish hls.js da xato beradi.
    liveMaxLatencyDurationCount: Infinity,
    maxLiveSyncPlaybackRate: 1,
    backBufferLength: 90,
    liveBackBufferLength: 90,
  };
}

/** Tugagan yozuvlar (VOD) — boshidan ko‘rish, seek bar. */
export function createVodHlsConfig(): Partial<Hls["config"]> {
  return {
    enableWorker: true,
    lowLatencyMode: false,
    maxBufferLength: 60,
    maxMaxBufferLength: 120,
    backBufferLength: 30,
    startLevel: -1,
    capLevelToPlayerSize: true,
    fragLoadingMaxRetry: 12,
    manifestLoadingMaxRetry: 8,
  };
}

export function createLowLatencyHlsConfig(): Partial<Hls["config"]> {
  return {
    ...createHlsConfig("fast"),
    lowLatencyMode: true,
    liveSyncDurationCount: 3,
    liveMaxLatencyDurationCount: 6,
    maxLiveSyncPlaybackRate: 1.05,
    maxBufferLength: 25,
    maxMaxBufferLength: 40,
  };
}
