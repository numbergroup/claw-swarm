"use client";

import {
  useState,
  useRef,
  useEffect,
  useLayoutEffect,
  useMemo,
  useCallback,
  type FormEvent,
} from "react";
import type { Message } from "@/lib/types";
import { MessageItem } from "./message-item";
import * as api from "@/lib/api";
import { ApiError } from "@/lib/api";

interface Props {
  spaceId: string;
  messages: Message[];
  hasMore: boolean;
  currentUserId: string;
  onLoadMore: () => void;
  loadingMore: boolean;
}

const INITIAL_VISIBLE_MESSAGES = 20;
const REVEAL_STEP = 20;
const TOP_THRESHOLD_PX = 80;

type LoadMode = "none" | "local" | "remote";

export function ChatPanel({
  spaceId,
  messages,
  hasMore,
  currentUserId,
  onLoadMore,
  loadingMore,
}: Props) {
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [visibleCount, setVisibleCount] = useState(INITIAL_VISIBLE_MESSAGES);

  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const wasNearBottom = useRef(true);
  const loadModeRef = useRef<LoadMode>("none");
  const remoteBaseLengthRef = useRef(0);
  const anchorRef = useRef<{ scrollHeight: number; scrollTop: number } | null>(null);
  const previousVisibleLastIdRef = useRef<string | null>(null);

  const visibleMessages = useMemo(() => {
    if (messages.length === 0) return [];
    const count = Math.min(visibleCount, messages.length);
    return messages.slice(-count);
  }, [messages, visibleCount]);

  const hiddenLoadedCount = Math.max(messages.length - visibleMessages.length, 0);

  const scrollToBottom = useCallback(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  const clearLoadState = useCallback(() => {
    loadModeRef.current = "none";
    remoteBaseLengthRef.current = 0;
    anchorRef.current = null;
  }, []);

  const captureAnchor = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;

    anchorRef.current = {
      scrollHeight: container.scrollHeight,
      scrollTop: container.scrollTop,
    };
  }, []);

  const restoreAnchor = useCallback(() => {
    const container = containerRef.current;
    const anchor = anchorRef.current;
    if (!container || !anchor) return;

    const delta = container.scrollHeight - anchor.scrollHeight;
    container.scrollTop = anchor.scrollTop + delta;
    clearLoadState();
  }, [clearLoadState]);

  const revealOlderMessages = useCallback(() => {
    if (loadModeRef.current !== "none") return;

    if (hiddenLoadedCount > 0) {
      captureAnchor();
      loadModeRef.current = "local";
      setVisibleCount((prev) => Math.min(prev + REVEAL_STEP, messages.length));
      return;
    }

    if (!hasMore || loadingMore) return;

    captureAnchor();
    loadModeRef.current = "remote";
    remoteBaseLengthRef.current = messages.length;
    onLoadMore();
  }, [captureAnchor, hasMore, hiddenLoadedCount, loadingMore, messages.length, onLoadMore]);

  useEffect(() => {
    setVisibleCount(INITIAL_VISIBLE_MESSAGES);
    clearLoadState();
    previousVisibleLastIdRef.current = null;
  }, [clearLoadState, spaceId]);

  useEffect(() => {
    if (messages.length === 0) {
      setVisibleCount(INITIAL_VISIBLE_MESSAGES);
      return;
    }

    setVisibleCount((prev) => (prev > messages.length ? messages.length : prev));
  }, [messages.length]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = container;
      wasNearBottom.current = scrollHeight - scrollTop - clientHeight < 100;

      if (scrollTop <= TOP_THRESHOLD_PX) {
        revealOlderMessages();
      }
    };

    container.addEventListener("scroll", handleScroll);
    return () => container.removeEventListener("scroll", handleScroll);
  }, [revealOlderMessages]);

  useLayoutEffect(() => {
    if (loadModeRef.current === "local") {
      restoreAnchor();
    }
  }, [restoreAnchor, visibleMessages.length]);

  useEffect(() => {
    if (loadModeRef.current !== "remote") return;
    if (loadingMore) return;

    if (messages.length > remoteBaseLengthRef.current) {
      const added = messages.length - remoteBaseLengthRef.current;
      loadModeRef.current = "local";
      setVisibleCount((prev) => Math.min(prev + added, messages.length));
      return;
    }

    clearLoadState();
  }, [clearLoadState, loadingMore, messages.length]);

  const lastVisibleMessageId =
    visibleMessages.length > 0 ? visibleMessages[visibleMessages.length - 1].id : null;

  useEffect(() => {
    if (!lastVisibleMessageId) return;

    const previousLastId = previousVisibleLastIdRef.current;
    previousVisibleLastIdRef.current = lastVisibleMessageId;

    if (previousLastId === lastVisibleMessageId) return;
    if (loadModeRef.current === "none" && wasNearBottom.current) {
      scrollToBottom();
    }
  }, [lastVisibleMessageId, scrollToBottom]);

  async function handleSend(e: FormEvent) {
    e.preventDefault();
    const content = input.trim();
    if (!content || sending) return;

    setSending(true);
    setInput("");

    try {
      await api.postMessage(spaceId, { content });
    } catch (err) {
      if (err instanceof ApiError) setInput(content);
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div ref={containerRef} className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {(hiddenLoadedCount > 0 || hasMore || loadingMore) && (
          <div className="text-center py-2 text-xs text-zinc-500">
            {loadingMore ? "Loading older messages..." : "Scroll up to load older messages"}
          </div>
        )}

        {visibleMessages.map((msg) => (
          <MessageItem
            key={msg.id}
            message={msg}
            isOwn={msg.senderId === currentUserId}
          />
        ))}
        <div ref={bottomRef} />
      </div>

      <form onSubmit={handleSend} className="border-t border-zinc-800 p-3 flex gap-2">
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Type a message..."
          className="flex-1 rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
        <button
          type="submit"
          disabled={sending || !input.trim()}
          className="rounded bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed px-4 py-2 text-sm font-medium text-white transition-colors"
        >
          Send
        </button>
      </form>
    </div>
  );
}
