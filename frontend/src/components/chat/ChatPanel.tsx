"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  chatWebSocketUrl,
  getChatHistory,
  type ChatMessage,
} from "@/lib/api/chat";
import {
  chatHistoryFailedMessage,
  chatLoginRequiredMessage,
  chatServerMessage,
  chatUnavailableMessage,
} from "@/lib/user-messages";
import { useAuthStore } from "@/stores/authStore";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface ChatPanelProps {
  streamId: string;
  live?: boolean;
}

type WsEvent = {
  type: string;
  id?: number;
  user_id?: string;
  username?: string;
  display_name?: string;
  content?: string;
  message_id?: number;
  ts?: number;
};

const MAX_WS_RETRIES = 6;

export function ChatPanel({ streamId, live = false }: ChatPanelProps) {
  const hydrated = useAuthStore((s) => s.hydrated);
  const accessToken = useAuthStore((s) => s.accessToken);
  const user = useAuthStore((s) => s.user);
  const canChat = hydrated && !!user && !!accessToken;
  const sessionPending = hydrated && !!user && !accessToken;
  const showLoginPrompt = hydrated && !user;

  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const listRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const retryRef = useRef(0);

  const loadHistory = useCallback(async (cursor?: number, prepend = false) => {
    const resp = await getChatHistory(streamId, cursor);
    setHasMore(resp.has_more);
    setMessages((prev) => {
      if (!prepend) return resp.data;
      const ids = new Set(prev.map((m) => m.id));
      const older = resp.data.filter((m) => !ids.has(m.id));
      return [...older, ...prev];
    });
    return resp.data[0]?.id;
  }, [streamId]);

  useEffect(() => {
    if (!hydrated) return;
    let cancelled = false;
    loadHistory()
      .then(() => {
        if (!cancelled) setError(null);
      })
      .catch(() => {
        if (!cancelled) setError(chatHistoryFailedMessage());
      });
    return () => {
      cancelled = true;
    };
  }, [loadHistory, hydrated]);

  useEffect(() => {
    if (!live || !hydrated) return;

    let ws: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let closed = false;

    const connect = () => {
      ws = new WebSocket(chatWebSocketUrl(streamId, accessToken));
      wsRef.current = ws;

      ws.onopen = () => {
        retryRef.current = 0;
        setConnected(true);
        setError(null);
      };

      ws.onclose = () => {
        setConnected(false);
        wsRef.current = null;
        if (closed) return;

        retryRef.current += 1;
        if (retryRef.current >= MAX_WS_RETRIES) {
          if (canChat) {
            setError(chatUnavailableMessage());
          }
          return;
        }

        const delay = Math.min(1000 * 2 ** retryRef.current, 30_000);
        reconnectTimer = setTimeout(connect, delay);
      };

      ws.onerror = () => {
        // onclose qayta ulanishni boshqaradi; mehmon uchun xato ko'rsatmaymiz.
      };

      ws.onmessage = (evt) => {
        try {
          const data = JSON.parse(evt.data) as WsEvent;
          if (data.type === "error") {
            if (canChat) {
              setError(chatServerMessage(data.content ?? ""));
            }
            return;
          }
          if (data.type === "delete" && data.message_id) {
            setMessages((prev) => prev.filter((m) => m.id !== data.message_id));
            return;
          }
          if (data.type === "message" && data.id) {
            const msg: ChatMessage = {
              id: data.id,
              stream_id: streamId,
              user_id: data.user_id,
              username: data.username ?? "",
              display_name: data.display_name ?? data.username ?? "",
              content: data.content ?? "",
              type: "text",
              created_at_unix: data.ts ?? Math.floor(Date.now() / 1000),
            };
            setMessages((prev) => {
              if (prev.some((m) => m.id === msg.id)) return prev;
              return [...prev, msg].slice(-200);
            });
          }
        } catch {
          /* ignore malformed */
        }
      };
    };

    connect();

    return () => {
      closed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      ws?.close();
      wsRef.current = null;
      setConnected(false);
    };
  }, [streamId, live, accessToken, hydrated, canChat]);

  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight });
  }, [messages.length]);

  const loadMore = useCallback(async () => {
    if (!hasMore || loadingMore || messages.length === 0) return;
    setLoadingMore(true);
    try {
      await loadHistory(messages[0].id, true);
    } finally {
      setLoadingMore(false);
    }
  }, [hasMore, loadingMore, messages, loadHistory]);

  const send = useCallback(() => {
    const text = input.trim();
    if (!text) return;
    if (!canChat) return;
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
    wsRef.current.send(JSON.stringify({ type: "message", content: text }));
    setInput("");
    setError(null);
  }, [input, canChat]);

  const loginHref = `/login?next=${encodeURIComponent(`/live/${streamId}`)}`;

  return (
    <div className="flex h-[480px] flex-col rounded-2xl border border-border bg-surface-1">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h3 className="font-semibold">Chat</h3>
        {live && canChat && (
          <span className={`text-xs ${connected ? "text-green-500" : "text-muted"}`}>
            {connected ? "Jonli" : "Qayta ulanmoqda…"}
          </span>
        )}
      </div>

      <div ref={listRef} className="flex-1 space-y-2 overflow-y-auto px-4 py-3">
        {hasMore && (
          <Button variant="ghost" size="sm" className="w-full" onClick={loadMore} disabled={loadingMore}>
            {loadingMore ? "Yuklanmoqda..." : "Eski xabarlar"}
          </Button>
        )}
        {messages.length === 0 && (
          <p className="text-sm text-muted">Hali xabar yo&apos;q. Birinchi bo&apos;ling!</p>
        )}
        {messages.map((m) => (
          <div key={m.id} className="text-sm">
            <span className="font-medium text-brand-secondary">{m.display_name || m.username}</span>
            <span className="text-muted">: </span>
            <span>{m.content}</span>
          </div>
        ))}
      </div>

      {error && canChat && (
        <p className="px-4 pb-1 text-xs text-amber-600 dark:text-amber-400">{error}</p>
      )}

      {canChat ? (
        <div className="flex gap-2 border-t border-border p-3">
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Xabar yozing..."
            disabled={!live}
            onKeyDown={(e) => e.key === "Enter" && send()}
          />
          <Button size="sm" onClick={send} disabled={!live || !input.trim()}>
            Yuborish
          </Button>
        </div>
      ) : sessionPending ? (
        <div className="border-t border-border p-4 text-center">
          <p className="text-sm text-muted">Chat tayyorlanmoqda…</p>
        </div>
      ) : showLoginPrompt ? (
        <div className="border-t border-border p-4 text-center">
          <p className="text-sm text-muted">{chatLoginRequiredMessage()}</p>
          <Link
            href={loginHref}
            className="mt-3 inline-flex rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90"
          >
            Kirish
          </Link>
        </div>
      ) : null}
    </div>
  );
}
