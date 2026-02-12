import type {
  CsBotRegistrationResponse,
  CsBotSpace,
  CsBotStatus,
  CsMessage,
  CsMessageListResponse,
  CsSpaceTask,
} from "./types.js";

const TAG = "[ClawSwarmClient]";

type Logger = (msg: string) => void;

export class ClawSwarmClient {
  private apiUrl: string;
  private token: string | null = null;
  private pollTimer: ReturnType<typeof setTimeout> | null = null;
  private shouldPoll = false;
  private log: Logger;

  constructor(apiUrl: string, log?: Logger) {
    this.apiUrl = apiUrl.replace(/\/+$/, "");
    this.log = log ?? ((msg: string) => console.log(`${TAG} ${msg}`));
  }

  // ── Auth ──────────────────────────────────────────────────────────

  setToken(token: string): void {
    this.token = token;
  }

  async refreshToken(): Promise<CsBotRegistrationResponse> {
    this.log("refreshToken: refreshing bot token");
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
    this.log("refreshToken: token refreshed successfully");
    return data;
  }

  // ── Bot Spaces ──────────────────────────────────────────────────────

  async getBotSpace(botSpaceId: string): Promise<CsBotSpace> {
    this.log(`getBotSpace: botSpaceId=${botSpaceId}`);
    const res = await fetch(`${this.apiUrl}/bot-spaces/${botSpaceId}`, {
      headers: { Authorization: `Bearer ${this.token}` },
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`getBotSpace failed (${res.status}): ${text}`);
    }
    this.log("getBotSpace: success");
    return res.json();
  }

  // ── Messages ──────────────────────────────────────────────────────

  async sendMessage(botSpaceId: string, content: string): Promise<CsMessage> {
    this.log(`sendMessage: botSpaceId=${botSpaceId} contentLength=${content.length}`);
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
    this.log("sendMessage: success");
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

  async getMessagesSince(
    botSpaceId: string,
    messageId: string,
    opts?: { limit?: number },
  ): Promise<CsMessageListResponse> {
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
    return res.json();
  }

  // ── Statuses ─────────────────────────────────────────────────────

  async updateBotStatus(
    botSpaceId: string,
    botId: string,
    status: string,
  ): Promise<CsBotStatus> {
    this.log(`updateBotStatus: botSpaceId=${botSpaceId} botId=${botId} status=${status}`);
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
      this.log(`updateBotStatus: got ${res.status}, refreshing token and retrying`);
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
      this.log("updateBotStatus: retry success");
      return retry.json();
    }

    if (!res.ok) {
      const text = await res.text();
      throw new Error(`updateBotStatus failed (${res.status}): ${text}`);
    }
    this.log("updateBotStatus: success");
    return res.json();
  }

  // ── Tasks ──────────────────────────────────────────────────────

  async listTasks(
    botSpaceId: string,
    opts?: { status?: string },
  ): Promise<CsSpaceTask[]> {
    const params = new URLSearchParams();
    if (opts?.status) params.set("status", opts.status);
    const qs = params.toString();
    const url = `${this.apiUrl}/bot-spaces/${botSpaceId}/tasks${qs ? `?${qs}` : ""}`;
    this.log(`listTasks: botSpaceId=${botSpaceId} status=${opts?.status ?? "(all)"}`);
    const res = await fetch(url, {
      headers: { Authorization: `Bearer ${this.token}` },
    });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`listTasks failed (${res.status}): ${text}`);
    }
    return res.json();
  }

  async getCurrentTask(botSpaceId: string): Promise<CsSpaceTask | null> {
    this.log(`getCurrentTask: botSpaceId=${botSpaceId}`);
    const res = await fetch(
      `${this.apiUrl}/bot-spaces/${botSpaceId}/tasks/current`,
      { headers: { Authorization: `Bearer ${this.token}` } },
    );
    if (res.status === 404) return null;
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`getCurrentTask failed (${res.status}): ${text}`);
    }
    return res.json();
  }

  async acceptTask(botSpaceId: string, taskId: string): Promise<CsSpaceTask> {
    this.log(`acceptTask: botSpaceId=${botSpaceId} taskId=${taskId}`);
    const res = await fetch(
      `${this.apiUrl}/bot-spaces/${botSpaceId}/tasks/${taskId}/accept`,
      {
        method: "POST",
        headers: { Authorization: `Bearer ${this.token}` },
      },
    );
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`acceptTask failed (${res.status}): ${text}`);
    }
    return res.json();
  }

  async completeTask(botSpaceId: string, taskId: string): Promise<CsSpaceTask> {
    this.log(`completeTask: botSpaceId=${botSpaceId} taskId=${taskId}`);
    const res = await fetch(
      `${this.apiUrl}/bot-spaces/${botSpaceId}/tasks/${taskId}/complete`,
      {
        method: "POST",
        headers: { Authorization: `Bearer ${this.token}` },
      },
    );
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`completeTask failed (${res.status}): ${text}`);
    }
    return res.json();
  }

  async blockTask(botSpaceId: string, taskId: string): Promise<CsSpaceTask> {
    this.log(`blockTask: botSpaceId=${botSpaceId} taskId=${taskId}`);
    const res = await fetch(
      `${this.apiUrl}/bot-spaces/${botSpaceId}/tasks/${taskId}/block`,
      {
        method: "POST",
        headers: { Authorization: `Bearer ${this.token}` },
      },
    );
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`blockTask failed (${res.status}): ${text}`);
    }
    return res.json();
  }

  async createTask(
    botSpaceId: string,
    name: string,
    description: string,
    botId?: string,
  ): Promise<CsSpaceTask> {
    this.log(`createTask: botSpaceId=${botSpaceId} name=${name}`);
    const body: Record<string, string> = { name, description };
    if (botId) body.botId = botId;
    const res = await fetch(
      `${this.apiUrl}/bot-spaces/${botSpaceId}/tasks`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.token}`,
        },
        body: JSON.stringify(body),
      },
    );
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`createTask failed (${res.status}): ${text}`);
    }
    return res.json();
  }

  async assignTask(
    botSpaceId: string,
    taskId: string,
    botId: string,
  ): Promise<CsSpaceTask> {
    this.log(`assignTask: botSpaceId=${botSpaceId} taskId=${taskId} botId=${botId}`);
    const res = await fetch(
      `${this.apiUrl}/bot-spaces/${botSpaceId}/tasks/${taskId}/assign`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.token}`,
        },
        body: JSON.stringify({ botId }),
      },
    );
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`assignTask failed (${res.status}): ${text}`);
    }
    return res.json();
  }

  // ── HTTP Polling ────────────────────────────────────────────────

  startPolling(
    botSpaceId: string,
    onMessage: (msg: CsMessage) => void,
    opts?: { intervalMs?: number },
  ): void {
    const intervalMs = opts?.intervalMs ?? 5000;
    this.log(`startPolling: botSpaceId=${botSpaceId} intervalMs=${intervalMs}`);
    this.shouldPoll = true;
    this.initPolling(botSpaceId, onMessage, intervalMs);
  }

  stopPolling(): void {
    this.log("stopPolling");
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
      this.log("initPolling: failed to seed cursor, starting from beginning");
    }
    this.log(`initPolling: cursor=${cursor ?? "(none)"}`);
    this.pollLoop(botSpaceId, cursor, onMessage, intervalMs);
  }

  private async pollLoop(
    botSpaceId: string,
    cursor: string | undefined,
    onMessage: (msg: CsMessage) => void,
    intervalMs: number,
  ): Promise<void> {
    if (!this.shouldPoll) return;

    try {
      if (cursor) {
        let hasMore = true;
        let currentCursor = cursor;
        while (hasMore) {
          const resp = await this.getMessagesSince(botSpaceId, currentCursor);
          if (resp.messages.length > 0) {
            this.log(`pollLoop: fetched ${resp.messages.length} messages hasMore=${resp.hasMore}`);
          }
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
          this.log(`pollLoop: seeded cursor=${cursor}`);
        }
      }
    } catch (err) {
      this.log(`pollLoop: error ${err}`);
      // Swallow errors and retry on next interval
    }

    if (!this.shouldPoll) return;
    this.pollTimer = setTimeout(
      () => this.pollLoop(botSpaceId, cursor, onMessage, intervalMs),
      intervalMs,
    );
  }
}
