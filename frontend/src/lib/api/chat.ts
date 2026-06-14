import { apiFetch, resolveApiUrl } from "@/lib/api/client";

export interface ChatMessage {
  id: number;
  stream_id: string;
  user_id?: string;
  username: string;
  display_name: string;
  content: string;
  type: string;
  created_at_unix: number;
}

export interface ChatHistoryResponse {
  data: ChatMessage[];
  has_more: boolean;
}

export function resolveWsUrl(): string {
  const api = resolveApiUrl();
  if (api.startsWith("https://")) return `wss://${api.slice(8)}`;
  if (api.startsWith("http://")) return `ws://${api.slice(7)}`;
  return "ws://localhost:8080";
}

export async function getChatHistory(
  streamId: string,
  cursor?: number,
  limit = 50,
): Promise<ChatHistoryResponse> {
  const params = new URLSearchParams({ limit: String(limit) });
  if (cursor) params.set("cursor", String(cursor));
  return apiFetch<ChatHistoryResponse>(`/v1/chat/${streamId}/history?${params}`);
}

export function chatWebSocketUrl(streamId: string, token?: string | null): string {
  const base = `${resolveWsUrl()}/v1/chat/${streamId}`;
  if (!token) return base;
  return `${base}?token=${encodeURIComponent(token)}`;
}
