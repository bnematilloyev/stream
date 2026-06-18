export type NetworkProfile = "fast" | "medium" | "slow" | "unknown";

type NetworkInformation = {
  downlink?: number;
  effectiveType?: string;
  saveData?: boolean;
};

function getConnection(): NetworkInformation | undefined {
  if (typeof navigator === "undefined") return undefined;
  const nav = navigator as Navigator & {
    connection?: NetworkInformation;
    mozConnection?: NetworkInformation;
    webkitConnection?: NetworkInformation;
  };
  return nav.connection ?? nav.mozConnection ?? nav.webkitConnection;
}

/** Navigator Network Information API (Chrome/Android) + effectiveType fallback. */
export function getNetworkProfile(): NetworkProfile {
  const conn = getConnection();
  if (!conn) return "unknown";

  if (conn.saveData) return "slow";

  const type = conn.effectiveType ?? "";
  if (type === "slow-2g" || type === "2g") return "slow";
  if (type === "3g") return "slow";

  const mbps = conn.downlink ?? 0;
  if (mbps > 0 && mbps < 1.5) return "slow";
  if (mbps > 0 && mbps < 4) return "medium";
  if (mbps >= 8 || type === "4g") return "fast";

  return "medium";
}

export function isMobileViewport(): boolean {
  if (typeof window === "undefined") return false;
  return window.matchMedia("(max-width: 768px)").matches;
}
