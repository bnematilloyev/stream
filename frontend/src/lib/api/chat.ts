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

/** Barcha chat tarixini (eski → yangi) yuklaydi — replay uchun. */
export async function getAllChatHistory(streamId: string): Promise<ChatMessage[]> {
  const all: ChatMessage[] = [];
  let cursor: number | undefined;

  for (;;) {
    const resp = await getChatHistory(streamId, cursor, 100);
    if (resp.data.length === 0) break;
    all.push(...resp.data);
    if (!resp.has_more) break;
    cursor = resp.data[resp.data.length - 1]?.id;
  }

  return all.sort((a, b) => a.id - b.id);
}

export function chatWebSocketUrl(streamId: string, token?: string | null): string {
  const base = `${resolveWsUrl()}/v1/chat/${streamId}`;
  if (!token) return base;
  return `${base}?token=${encodeURIComponent(token)}`;
}
