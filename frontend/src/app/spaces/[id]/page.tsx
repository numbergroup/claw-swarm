"use client";

import { useState, useEffect, useCallback, useRef, use } from "react";
import { AuthGuard } from "@/components/auth-guard";
import { Header } from "@/components/header";
import { useAuth } from "@/lib/auth-context";
import { useWebSocket } from "@/lib/use-websocket";
import * as api from "@/lib/api";
import type {
  BotSpace,
  Bot,
  BotStatus,
  Message,
  Summary,
  SpaceMemberWithUser,
  InviteCode,
} from "@/lib/types";
import { ChatPanel } from "@/components/space/chat-panel";
import { BotListPanel } from "@/components/space/bot-list-panel";
import { SummaryPanel } from "@/components/space/summary-panel";
import { MembersPanel } from "@/components/space/members-panel";
import { BotStatusPanel } from "@/components/space/bot-status-panel";
import { JoinCodesPanel } from "@/components/space/join-codes-panel";
import { InviteCodesPanel } from "@/components/space/invite-codes-panel";
import { SpaceHeader } from "@/components/space/space-header";

type LoadState = "loading" | "ready" | "forbidden" | "notFound" | "error";

function compareMessages(a: Message, b: Message): number {
  if (a.createdAt === b.createdAt) {
    return a.id.localeCompare(b.id);
  }
  return a.createdAt.localeCompare(b.createdAt);
}

function normalizeMessages(messages: Message[]): Message[] {
  return [...messages].sort(compareMessages);
}

function mergeMessages(existing: Message[], incoming: Message[]): Message[] {
  if (incoming.length === 0) return existing;

  const map = new Map(existing.map((msg) => [msg.id, msg]));
  for (const msg of incoming) {
    map.set(msg.id, msg);
  }

  return normalizeMessages(Array.from(map.values()));
}

function SpaceWorkspace({ spaceId }: { spaceId: string }) {
  const { user, token } = useAuth();
  const [space, setSpace] = useState<BotSpace | null>(null);
  const [bots, setBots] = useState<Bot[]>([]);
  const [statuses, setStatuses] = useState<BotStatus[]>([]);
  const [messages, setMessages] = useState<Message[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [members, setMembers] = useState<SpaceMemberWithUser[]>([]);
  const [inviteCodes, setInviteCodes] = useState<InviteCode[]>([]);
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const latestMessageIdRef = useRef<string | null>(null);
  const syncingSinceReconnectRef = useRef(false);

  const isOwner = space?.ownerId === user?.id;

  useEffect(() => {
    latestMessageIdRef.current = messages.length > 0 ? messages[messages.length - 1].id : null;
  }, [messages]);

  const fetchInitialData = useCallback(async () => {
    setLoadState("loading");
    setLoadError(null);
    setLoadingMore(false);
    setSpace(null);
    setBots([]);
    setStatuses([]);
    setMessages([]);
    setHasMore(false);
    setSummary(null);
    setMembers([]);
    setInviteCodes([]);

    try {
      const [spaceData, botsData, messagesData, membersData] = await Promise.all([
        api.getBotSpace(spaceId),
        api.listBots(spaceId),
        api.listMessages(spaceId, undefined, 30),
        api.listMembers(spaceId),
      ]);

      setSpace(spaceData);
      setBots(botsData);
      setMessages(normalizeMessages(messagesData.messages));
      setHasMore(messagesData.hasMore);
      setMembers(membersData);
      setLoadState("ready");

      api.listStatuses(spaceId).then(setStatuses).catch(() => {});
      api.getSummary(spaceId).then(setSummary).catch(() => {});

      if (spaceData.ownerId === user?.id) {
        api.listInviteCodes(spaceId).then(setInviteCodes).catch(() => {});
      }
    } catch (err) {
      if (err instanceof api.ApiError) {
        if (err.status === 404) {
          setLoadState("notFound");
          return;
        }
        if (err.status === 403) {
          setLoadState("forbidden");
          return;
        }
        setLoadError(err.message);
      }
      setLoadState("error");
    }
  }, [spaceId, user?.id]);

  useEffect(() => {
    void fetchInitialData();
  }, [fetchInitialData]);

  useEffect(() => {
    if (loadState !== "ready") return;

    const interval = setInterval(() => {
      api.listBots(spaceId).then(setBots).catch(() => {});
      api.listStatuses(spaceId).then(setStatuses).catch(() => {});
      api.getSummary(spaceId).then(setSummary).catch(() => {});
    }, 15000);

    return () => clearInterval(interval);
  }, [loadState, spaceId]);

  const handleWsMessage = useCallback((msg: Message) => {
    setMessages((prev) => mergeMessages(prev, [msg]));
  }, []);

  const syncMessagesSinceReconnect = useCallback(async () => {
    const startCursor = latestMessageIdRef.current;
    if (!startCursor || syncingSinceReconnectRef.current) return;

    syncingSinceReconnectRef.current = true;

    try {
      let cursor = startCursor;
      for (;;) {
        const res = await api.getMessagesSince(spaceId, cursor, 30);
        const normalized = normalizeMessages(res.messages);
        if (normalized.length === 0) break;

        setMessages((prev) => mergeMessages(prev, normalized));
        cursor = normalized[normalized.length - 1].id;

        if (!res.hasMore) break;
      }
    } catch {
      // Best effort backfill; websocket stream remains active.
    } finally {
      syncingSinceReconnectRef.current = false;
    }
  }, [spaceId]);

  useWebSocket({
    spaceId,
    token: loadState === "ready" ? token : null,
    onMessage: handleWsMessage,
    onOpen: (isReconnect) => {
      if (!isReconnect) return;
      void syncMessagesSinceReconnect();
    },
  });

  const handleLoadMore = useCallback(async () => {
    if (loadingMore || !hasMore || messages.length === 0) return;

    setLoadingMore(true);
    try {
      const oldest = messages[0];
      const res = await api.listMessages(spaceId, oldest.id, 30);
      setMessages((prev) => mergeMessages(prev, normalizeMessages(res.messages)));
      setHasMore(res.hasMore);
    } catch {
      // Keep current state when fetching older history fails.
    } finally {
      setLoadingMore(false);
    }
  }, [hasMore, loadingMore, messages, spaceId]);

  const refreshBots = useCallback(() => {
    api.listBots(spaceId).then(setBots).catch(() => {});
    api.listStatuses(spaceId).then(setStatuses).catch(() => {});
  }, [spaceId]);

  const refreshMembers = useCallback(() => {
    api.listMembers(spaceId).then(setMembers).catch(() => {});
  }, [spaceId]);

  const refreshInviteCodes = useCallback(() => {
    api.listInviteCodes(spaceId).then(setInviteCodes).catch(() => {});
  }, [spaceId]);

  if (loadState === "loading") {
    return (
      <>
        <Header />
        <div className="flex items-center justify-center h-[calc(100vh-3.5rem)]">
          <div className="text-zinc-400">Loading space...</div>
        </div>
      </>
    );
  }

  if (loadState === "notFound") {
    return (
      <>
        <Header />
        <div className="flex items-center justify-center h-[calc(100vh-3.5rem)]">
          <div className="text-zinc-400">Space not found</div>
        </div>
      </>
    );
  }

  if (loadState === "forbidden") {
    return (
      <>
        <Header />
        <div className="flex flex-col items-center justify-center h-[calc(100vh-3.5rem)] gap-3">
          <div className="text-zinc-300">You do not have access to this space.</div>
          <button
            onClick={() => void fetchInitialData()}
            className="rounded border border-zinc-700 px-3 py-1.5 text-sm text-zinc-300 hover:border-zinc-500 transition-colors"
          >
            Retry
          </button>
        </div>
      </>
    );
  }

  if (loadState === "error" || !space) {
    return (
      <>
        <Header />
        <div className="flex flex-col items-center justify-center h-[calc(100vh-3.5rem)] gap-3 px-4 text-center">
          <div className="text-zinc-300">Failed to load this space.</div>
          {loadError && <div className="text-sm text-zinc-500">{loadError}</div>}
          <button
            onClick={() => void fetchInitialData()}
            className="rounded bg-blue-600 hover:bg-blue-500 px-4 py-2 text-sm font-medium text-white transition-colors"
          >
            Retry
          </button>
        </div>
      </>
    );
  }

  return (
    <>
      <Header />
      <div className="h-[calc(100vh-3.5rem)] flex">
        <div className="w-64 border-r border-zinc-800 overflow-y-auto py-4 space-y-6 shrink-0 hidden lg:block">
          <BotListPanel
            bots={bots}
            statuses={statuses}
            isOwner={isOwner}
            spaceId={spaceId}
            onUpdated={refreshBots}
          />
          <BotStatusPanel bots={bots} statuses={statuses} />
          <MembersPanel
            members={members}
            isOwner={isOwner}
            spaceId={spaceId}
            currentUserId={user?.id ?? ""}
            onUpdated={refreshMembers}
          />
        </div>

        <div className="flex-1 flex flex-col min-w-0">
          <div className="border-b border-zinc-800 px-4 py-3">
            <SpaceHeader space={space} isOwner={isOwner} onUpdated={setSpace} />
          </div>
          <div className="flex-1 min-h-0">
            <ChatPanel
              spaceId={spaceId}
              messages={messages}
              hasMore={hasMore}
              currentUserId={user?.id ?? ""}
              onLoadMore={handleLoadMore}
              loadingMore={loadingMore}
            />
          </div>
        </div>

        <div className="w-72 border-l border-zinc-800 overflow-y-auto py-4 space-y-6 shrink-0 hidden xl:block">
          <SummaryPanel summary={summary} />
          {isOwner && (
            <>
              <JoinCodesPanel space={space} onUpdated={setSpace} />
              <InviteCodesPanel
                spaceId={spaceId}
                codes={inviteCodes}
                onUpdated={refreshInviteCodes}
              />
            </>
          )}
        </div>
      </div>
    </>
  );
}

export default function SpacePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <AuthGuard>
      <SpaceWorkspace spaceId={id} />
    </AuthGuard>
  );
}
