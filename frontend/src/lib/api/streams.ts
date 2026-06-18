import { apiFetch, ApiClientError } from "./client";
import type { PaginatedStreams, Playback, Stream } from "@/types";

export async function getLiveStreams(page = 1, limit = 24) {
  return apiFetch<PaginatedStreams>(
    `/v1/streams/live?page=${page}&limit=${limit}`,
  );
}

export async function getStream(id: string) {
  return apiFetch<Stream>(`/v1/streams/${id}`);
}

export async function getPlayback(id: string) {
  return apiFetch<Playback>(`/v1/streams/${id}/playback`);
}

/** Jonli yoki yozilgan stream playback. */
export async function getStreamPlayback(
  id: string,
  signal?: AbortSignal,
  opts?: { warmup?: boolean },
) {
  const maxAttempts = opts?.warmup === false ? 1 : 40;
  let lastError: unknown;

  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    if (signal?.aborted) {
      throw new DOMException("Aborted", "AbortError");
    }
    try {
      return await getPlayback(id);
    } catch (e) {
      lastError = e;
      if (
        e instanceof ApiClientError &&
        e.status === 404 &&
        attempt < maxAttempts - 1
      ) {
        await new Promise((r) => setTimeout(r, 2000));
        continue;
      }
      throw e;
    }
  }

  throw lastError;
}

/** @deprecated use getStreamPlayback */
export async function getPlaybackWhenLive(
  id: string,
  signal?: AbortSignal,
): Promise<Playback> {
  return getStreamPlayback(id, signal);
}

export async function recordViewerHeartbeat(streamId: string, sessionId: string) {
  return apiFetch<{ stream_id: string; concurrent: number; unique: number }>(
    `/v1/streams/${streamId}/heartbeat`,
    {
      method: "POST",
      body: JSON.stringify({ session_id: sessionId }),
    },
  );
}

export async function getChannelStreams(
  slug: string,
  page = 1,
  limit = 20,
  status?: string,
) {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (status) params.set("status", status);
  return apiFetch<PaginatedStreams>(
    `/v1/channels/${slug}/streams?${params}`,
  );
}

export async function createStream(data: {
  channel_slug: string;
  title: string;
  description?: string;
  visibility?: string;
  latency_mode?: string;
  ingest_protocol?: string;
}) {
  return apiFetch<Stream>("/v1/streams", {
    method: "POST",
    auth: true,
    body: JSON.stringify(data),
  });
}

export async function startStream(id: string) {
  return apiFetch<Stream>(`/v1/streams/${id}/start`, {
    method: "POST",
    auth: true,
  });
}

export async function endStream(id: string) {
  return apiFetch<Stream>(`/v1/streams/${id}/end`, {
    method: "POST",
    auth: true,
  });
}

export async function endChannelLiveStreams(channelSlug: string) {
  const { data } = await getChannelStreams(channelSlug, 1, 20, "live");
  await Promise.all(data.map((s) => endStream(s.id).catch(() => undefined)));
}
