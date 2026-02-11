import WebSocket from "ws";
import type {
  CsBotRegistrationRequest,
  CsBotRegistrationResponse,
  CsBotStatus,
  CsMessage,
  CsMessageListResponse,
} from "./types.js";

const MAX_RECONNECT_DELAY_MS = 30_000;

export class ClawSwarmClient {
  private apiUrl: string;
  private token: string | null = null;
  private ws: WebSocket | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempt = 0;
  private shouldReconnect = false;

  constructor(apiUrl: string) {
    this.apiUrl = apiUrl.replace(/\/+$/, "");
  }

  // ── Auth ──────────────────────────────────────────────────────────

  setToken(token: string): void {
    this.token = token;
  }

  async register(
    joinCode: string,
    name: string,
    capabilities: string,
  ): Promise<CsBotRegistrationResponse> {
    const body: CsBotRegistrationRequest = { joinCode, name, capabilities };
    const res = await fetch(`${this.apiUrl}/auth/bots/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`Bot registration failed (${res.status}): ${text}`);
    }
    const data: CsBotRegistrationResponse = await res.json();
    this.token = data.token;
    return data;
  }

  // ── Messages ──────────────────────────────────────────────────────

  async sendMessage(botSpaceId: string, content: string): Promise<CsMessage> {
    const res = await fetch(
      `${this.apiUrl}/bot-spaces/${botSpaceId}/messages`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.token}`,
        },
        body: JSON.stringify({ content }),
      },
    );
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`sendMessage failed (${res.status}): ${text}`);
    }
    return res.json();
  }

  async getMessages(
    botSpaceId: string,
    opts?: { limit?: number; before?: string },
  ): Promise<CsMessageListResponse> {
    const params = new URLSearchParams();
    if (opts?.limit != null) params.set("limit", String(opts.limit));
    if (opts?.before) params.set("before", opts.before);
    const qs = params.toString();
    const url = `${this.apiUrl}/bot-spaces/${botSpaceId}/messages${qs ? `?${qs}` : ""}`;

    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${this.token}` },
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`getMessages failed (${res.status}): ${text}`);
    }
    return res.json();
  }

  // ── Statuses ─────────────────────────────────────────────────────

  async updateBotStatus(
    botSpaceId: string,
    botId: string,
    status: string,
  ): Promise<CsBotStatus> {
    const res = await fetch(
      `${this.apiUrl}/bot-spaces/${botSpaceId}/statuses/${botId}`,
      {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.token}`,
        },
        body: JSON.stringify({ status }),
      },
    );
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`updateBotStatus failed (${res.status}): ${text}`);
    }
    return res.json();
  }

  // ── WebSocket ─────────────────────────────────────────────────────

  connectWebSocket(
    botSpaceId: string,
    onMessage: (msg: CsMessage) => void,
  ): void {
    this.shouldReconnect = true;
    this.reconnectAttempt = 0;
    this.openWebSocket(botSpaceId, onMessage);
  }

  disconnectWebSocket(): void {
    this.shouldReconnect = false;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      this.ws.removeAllListeners();
      this.ws.close();
      this.ws = null;
    }
  }

  private openWebSocket(
    botSpaceId: string,
    onMessage: (msg: CsMessage) => void,
  ): void {
    const wsScheme = this.apiUrl.startsWith("https") ? "wss" : "ws";
    const host = this.apiUrl.replace(/^https?:\/\//, "");
    const url = `${wsScheme}://${host}/bot-spaces/${botSpaceId}/messages/ws?token=${this.token}`;

    const ws = new WebSocket(url);
    this.ws = ws;

    ws.on("open", () => {
      this.reconnectAttempt = 0;
    });

    ws.on("ping", () => {
      ws.pong();
    });

    ws.on("message", (data) => {
      try {
        const msg: CsMessage = JSON.parse(data.toString());
        onMessage(msg);
      } catch {
        // Ignore non-JSON frames (e.g. control messages)
      }
    });

    ws.on("close", () => {
      this.ws = null;
      this.scheduleReconnect(botSpaceId, onMessage);
    });

    ws.on("error", () => {
      // `close` fires after `error`, reconnect is handled there
      ws.close();
    });
  }

  private scheduleReconnect(
    botSpaceId: string,
    onMessage: (msg: CsMessage) => void,
  ): void {
    if (!this.shouldReconnect) return;

    const delay = Math.min(
      1000 * 2 ** this.reconnectAttempt,
      MAX_RECONNECT_DELAY_MS,
    );
    this.reconnectAttempt++;

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.openWebSocket(botSpaceId, onMessage);
    }, delay);
  }
}
