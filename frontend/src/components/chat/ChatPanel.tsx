"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  chatWebSocketUrl,
  getAllChatHistory,
  getChatHistory,
  type ChatMessage,
} from "@/lib/api/chat";
import {
  chatHistoryFailedMessage,
  chatLoginRequiredMessage,
  chatServerMessage,
  chatUnavailableMessage,
} from "@/lib/user-messages";
import { refreshAccessToken } from "@/lib/api/client";
import { useAuthStore } from "@/stores/authStore";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { FeaturedProduct } from "@/lib/api/featured";

interface ChatPanelProps {
  streamId: string;
  live?: boolean;
  replay?: boolean;
  streamStartedAtUnix?: number;
  playbackSec?: number;
  /** Efir egasi mahsulot ajratganda real-vaqtda chaqiriladi (null = bekor qilindi). */
  onFeaturedProduct?: (product: FeaturedProduct | null) => void;
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
  product?: FeaturedProduct | null;
};

const MAX_WS_RETRIES = 6;

function messageStreamOffset(msg: ChatMessage, streamStartedAtUnix: number): number {
  if (streamStartedAtUnix <= 0) return 0;
  return Math.max(0, msg.created_at_unix - streamStartedAtUnix);
}

export function ChatPanel({
  streamId,
  live = false,
  replay = false,
  streamStartedAtUnix = 0,
  playbackSec = 0,
  onFeaturedProduct,
}: ChatPanelProps) {
  const hydrated = useAuthStore((s) => s.hydrated);
  const accessToken = useAuthStore((s) => s.accessToken);
  const user = useAuthStore((s) => s.user);
  const canChat = hydrated && !!user && !!accessToken && live && !replay;
  const sessionPending = hydrated && !!user && !accessToken && live && !replay;
  const showLoginPrompt = hydrated && !user && live && !replay;

  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [replayMessages, setReplayMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [replayLoading, setReplayLoading] = useState(false);
  const listRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const retryRef = useRef(0);
  const accessTokenRef = useRef(accessToken);
  const canChatRef = useRef(canChat);
  const featuredCbRef = useRef(onFeaturedProduct);

  useEffect(() => {
    featuredCbRef.current = onFeaturedProduct;
  }, [onFeaturedProduct]);

  useEffect(() => {
    accessTokenRef.current = accessToken;
  }, [accessToken]);

  useEffect(() => {
    canChatRef.current = canChat;
  }, [canChat]);

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
    if (!hydrated || replay) return;
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
  }, [loadHistory, hydrated, replay]);

  useEffect(() => {
    if (!hydrated || !replay) return;
    let cancelled = false;
    setReplayLoading(true);
    getAllChatHistory(streamId)
      .then((list) => {
        if (!cancelled) {
          setReplayMessages(list);
          setError(null);
        }
      })
      .catch(() => {
        if (!cancelled) setError(chatHistoryFailedMessage());
      })
      .finally(() => {
        if (!cancelled) setReplayLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [streamId, hydrated, replay]);

  useEffect(() => {
    if (!live || !hydrated || replay) return;

    let ws: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let closed = false;

    const connect = async () => {
      if (closed) return;

      let token = accessTokenRef.current;
      if (user && !token) {
        token = await refreshAccessToken();
        if (token) accessTokenRef.current = token;
      }

      ws = new WebSocket(chatWebSocketUrl(streamId, token));
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
          if (canChatRef.current) {
            setError(chatUnavailableMessage());
          }
          return;
        }

        const delay = Math.min(1000 * 2 ** retryRef.current, 30_000);
        reconnectTimer = setTimeout(() => {
          void connect();
        }, delay);
      };

      ws.onerror = () => {
        // onclose qayta ulanishni boshqaradi
      };

      ws.onmessage = (evt) => {
        try {
          const data = JSON.parse(evt.data) as WsEvent;
          if (data.type === "error") {
            if (canChatRef.current) {
              setError(chatServerMessage(data.content ?? ""));
            }
            return;
          }
          if (data.type === "delete" && data.message_id) {
            setMessages((prev) => prev.filter((m) => m.id !== data.message_id));
            return;
          }
          if (data.type === "featured_product") {
            featuredCbRef.current?.(data.product ?? null);
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

    void connect();

    return () => {
      closed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      ws?.close();
      wsRef.current = null;
      setConnected(false);
    };
  }, [streamId, live, hydrated, user, replay]);

  const displayMessages = useMemo(() => {
    if (!replay) return messages;
    if (streamStartedAtUnix <= 0) return replayMessages;
    return replayMessages.filter(
      (m) => messageStreamOffset(m, streamStartedAtUnix) <= playbackSec + 0.25,
    );
  }, [replay, messages, replayMessages, streamStartedAtUnix, playbackSec]);

  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight });
  }, [displayMessages.length]);

  const loadMore = useCallback(async () => {
    if (!hasMore || loadingMore || messages.length === 0 || replay) return;
    setLoadingMore(true);
    try {
      await loadHistory(messages[0].id, true);
    } finally {
      setLoadingMore(false);
    }
  }, [hasMore, loadingMore, messages, loadHistory, replay]);

  const send = useCallback(() => {
    const text = input.trim();
    if (!text) return;
    if (!canChat) return;
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      setError("Chat hali ulanmagan. Bir oz kuting yoki sahifani yangilang.");
      return;
    }
    wsRef.current.send(JSON.stringify({ type: "message", content: text }));
    setInput("");
    setError(null);
  }, [input, canChat]);

  const loginHref = `/login?next=${encodeURIComponent(`/live/${streamId}`)}`;

  return (
    <div className="flex h-[480px] flex-col rounded-2xl border border-border bg-surface-1">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h3 className="font-semibold">Chat</h3>
        {replay ? (
          <span className="text-xs text-muted">Chat takrori</span>
        ) : (
          live &&
          canChat && (
            <span className={`text-xs ${connected ? "text-green-500" : "text-muted"}`}>
              {connected ? "Jonli" : "Qayta ulanmoqda…"}
            </span>
          )
        )}
      </div>

      <div ref={listRef} className="flex-1 space-y-2 overflow-y-auto px-4 py-3">
        {!replay && hasMore && (
          <Button variant="ghost" size="sm" className="w-full" onClick={loadMore} disabled={loadingMore}>
            {loadingMore ? "Yuklanmoqda..." : "Eski xabarlar"}
          </Button>
        )}
        {replayLoading && (
          <p className="text-sm text-muted">Chat yuklanmoqda…</p>
        )}
        {!replayLoading && displayMessages.length === 0 && (
          <p className="text-sm text-muted">
            {replay ? "Bu vaqtga chat xabari yo'q." : "Hali xabar yo'q. Birinchi bo'ling!"}
          </p>
        )}
        {displayMessages.map((m) => (
          <div key={m.id} className="text-sm leading-snug">
            <span className="font-medium text-brand-secondary">
              {m.display_name || m.username}
            </span>
            <span className="text-muted">: </span>
            <span>{m.content}</span>
          </div>
        ))}
      </div>

      {error && (canChat || replay) && (
        <p className="px-4 pb-1 text-xs text-amber-600 dark:text-amber-400">{error}</p>
      )}

      {replay ? (
        <div className="border-t border-border p-4 text-center">
          <p className="text-sm text-muted">
            Efir tugagan — chat faqat ko&apos;rish uchun
          </p>
        </div>
      ) : canChat ? (
        <div className="flex gap-2 border-t border-border p-3">
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Xabar yozing..."
            disabled={!live}
            onKeyDown={(e) => e.key === "Enter" && send()}
          />
          <Button
            size="sm"
            onClick={send}
            disabled={!live || !connected || !input.trim()}
          >
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
