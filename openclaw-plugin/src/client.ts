import type {
  CsBotRegistrationResponse,
  CsBotStatus,
  CsMessage,
  CsMessageListResponse,
} from "./types.js";

const TAG = "[ClawSwarmClient]";

export class ClawSwarmClient {
  private apiUrl: string;
  private token: string | null = null;
  private pollTimer: ReturnType<typeof setTimeout> | null = null;
  private shouldPoll = false;

  constructor(apiUrl: string) {
    this.apiUrl = apiUrl.replace(/\/+$/, "");
  }

  // ── Auth ──────────────────────────────────────────────────────────

  setToken(token: string): void {
    this.token = token;
  }

  async refreshToken(): Promise<CsBotRegistrationResponse> {
    console.debug(TAG, "refreshToken: refreshing bot token");
    const res = await fetch(`${this.apiUrl}/auth/bots/refresh`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${this.token}`,
      },
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`Token refresh failed (${res.status}): ${text}`);
    }
    const data: CsBotRegistrationResponse = await res.json();
    this.token = data.token;
    console.debug(TAG, "refreshToken: token refreshed successfully");
    return data;
  }

  // ── Messages ──────────────────────────────────────────────────────

  async sendMessage(botSpaceId: string, content: string): Promise<CsMessage> {
    console.debug(TAG, "sendMessage: botSpaceId=%s contentLength=%d", botSpaceId, content.length);
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
    console.debug(TAG, "sendMessage: success");
    return res.json();
  }

  async getMessages(
    botSpaceId: string,
    opts?: { limit?: number; before?: string },
  ): Promise<CsMessageListResponse> {
    console.debug(TAG, "getMessages: botSpaceId=%s opts=%o", botSpaceId, opts);
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
    console.debug(TAG, "getMessages: success");
    return res.json();
  }

  async getMessagesSince(
    botSpaceId: string,
    messageId: string,
    opts?: { limit?: number },
  ): Promise<CsMessageListResponse> {
    console.debug(TAG, "getMessagesSince: botSpaceId=%s messageId=%s opts=%o", botSpaceId, messageId, opts);
    const params = new URLSearchParams();
    if (opts?.limit != null) params.set("limit", String(opts.limit));
    const qs = params.toString();
    const url = `${this.apiUrl}/bot-spaces/${botSpaceId}/messages/since/${messageId}${qs ? `?${qs}` : ""}`;
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${this.token}` },
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`getMessagesSince failed (${res.status}): ${text}`);
    }
    console.debug(TAG, "getMessagesSince: success");
    return res.json();
  }

  // ── Statuses ─────────────────────────────────────────────────────

  async updateBotStatus(
    botSpaceId: string,
    botId: string,
    status: string,
  ): Promise<CsBotStatus> {
    console.debug(TAG, "updateBotStatus: botSpaceId=%s botId=%s status=%s", botSpaceId, botId, status);
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

    if (res.status === 401 || res.status === 403) {
      console.debug(TAG, "updateBotStatus: got %d, refreshing token and retrying", res.status);
      await this.refreshToken();
      const retry = await fetch(
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
      if (!retry.ok) {
        const text = await retry.text();
        throw new Error(`updateBotStatus failed (${retry.status}): ${text}`);
      }
      console.debug(TAG, "updateBotStatus: retry success");
      return retry.json();
    }

    if (!res.ok) {
      const text = await res.text();
      throw new Error(`updateBotStatus failed (${res.status}): ${text}`);
    }
    console.debug(TAG, "updateBotStatus: success");
    return res.json();
  }

  // ── HTTP Polling ────────────────────────────────────────────────

  startPolling(
    botSpaceId: string,
    onMessage: (msg: CsMessage) => void,
    opts?: { intervalMs?: number },
  ): void {
    const intervalMs = opts?.intervalMs ?? 5000;
    console.debug(TAG, "startPolling: botSpaceId=%s intervalMs=%d", botSpaceId, intervalMs);
    this.shouldPoll = true;
    this.initPolling(botSpaceId, onMessage, intervalMs);
  }

  stopPolling(): void {
    console.debug(TAG, "stopPolling");
    this.shouldPoll = false;
    if (this.pollTimer) {
      clearTimeout(this.pollTimer);
      this.pollTimer = null;
    }
  }

  private async initPolling(
    botSpaceId: string,
    onMessage: (msg: CsMessage) => void,
    intervalMs: number,
  ): Promise<void> {
    let cursor: string | undefined;
    try {
      const { messages } = await this.getMessages(botSpaceId, { limit: 1 });
      if (messages.length > 0) {
        cursor = messages[0].id;
      }
    } catch {
      console.debug(TAG, "initPolling: failed to seed cursor, starting from beginning");
    }
    console.debug(TAG, "initPolling: cursor=%s", cursor ?? "(none)");
    this.pollLoop(botSpaceId, cursor, onMessage, intervalMs);
  }

  private async pollLoop(
    botSpaceId: string,
    cursor: string | undefined,
    onMessage: (msg: CsMessage) => void,
    intervalMs: number,
  ): Promise<void> {
    if (!this.shouldPoll) return;
    console.debug(TAG, "pollLoop: tick cursor=%s", cursor ?? "(none)");

    try {
      if (cursor) {
        let hasMore = true;
        let currentCursor = cursor;
        while (hasMore) {
          const resp = await this.getMessagesSince(botSpaceId, currentCursor);
          console.debug(TAG, "pollLoop: fetched %d messages hasMore=%s", resp.messages.length, resp.hasMore);
          for (const msg of resp.messages) {
            onMessage(msg);
            currentCursor = msg.id;
          }
          hasMore = resp.hasMore;
        }
        cursor = currentCursor;
      } else {
        // No cursor yet — fetch latest to establish one
        const { messages } = await this.getMessages(botSpaceId, { limit: 1 });
        if (messages.length > 0) {
          cursor = messages[0].id;
          console.debug(TAG, "pollLoop: seeded cursor=%s", cursor);
        }
      }
    } catch (err) {
      console.debug(TAG, "pollLoop: error %s", err);
      // Swallow errors and retry on next interval
    }

    if (!this.shouldPoll) return;
    this.pollTimer = setTimeout(
      () => this.pollLoop(botSpaceId, cursor, onMessage, intervalMs),
      intervalMs,
    );
  }
}
