"use client";

import { useEffect, useRef } from "react";
import type { Message } from "./types";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

function getWsUrl(spaceId: string, token: string): string {
  const base = API_URL.replace(/^http/, "ws");
  return `${base}/bot-spaces/${spaceId}/messages/ws?token=${token}`;
}

interface UseWebSocketOptions {
  spaceId: string;
  token: string | null;
  onMessage: (msg: Message) => void;
  onOpen?: (isReconnect: boolean) => void;
}

export function useWebSocket({ spaceId, token, onMessage, onOpen }: UseWebSocketOptions) {
  const onMessageRef = useRef(onMessage);
  const onOpenRef = useRef(onOpen);

  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    onOpenRef.current = onOpen;
  }, [onOpen]);

  useEffect(() => {
    if (!token || !spaceId) return;

    let ws: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let reconnectDelay = 1000;
    let mounted = true;
    let hadSuccessfulConnection = false;

    const clearReconnectTimer = () => {
      if (!reconnectTimer) return;
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    };

    const connect = () => {
      if (!mounted) return;

      const socket = new WebSocket(getWsUrl(spaceId, token));
      ws = socket;

      socket.onopen = () => {
        onOpenRef.current?.(hadSuccessfulConnection);
        hadSuccessfulConnection = true;
        reconnectDelay = 1000;
      };

      socket.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data) as Message;
          onMessageRef.current(msg);
        } catch {
          // Ignore malformed websocket payloads.
        }
      };

      socket.onclose = () => {
        if (ws !== socket) return;
        ws = null;
        if (!mounted) return;

        reconnectTimer = setTimeout(() => {
          reconnectDelay = Math.min(reconnectDelay * 2, 30000);
          connect();
        }, reconnectDelay);
      };

      socket.onerror = () => {
        if (ws !== socket) return;
        socket.close();
      };
    };

    connect();

    return () => {
      mounted = false;
      clearReconnectTimer();

      if (ws) {
        ws.onclose = null;
        ws.onerror = null;
        ws.close();
      }
    };
  }, [spaceId, token]);
}
