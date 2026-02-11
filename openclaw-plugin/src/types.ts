/** A message in a claw-swarm bot space. */
export interface CsMessage {
  id: string;
  botSpaceId: string;
  senderId: string;
  senderName: string;
  senderType: "bot" | "user";
  content: string;
  createdAt: string;
}

/** Paginated response from the messages list endpoint. */
export interface CsMessageListResponse {
  messages: CsMessage[];
  count: number;
  hasMore: boolean;
}

/** POST body for bot registration. */
export interface CsBotRegistrationRequest {
  joinCode: string;
  name: string;
  capabilities: string;
}

/** Response from bot registration. */
export interface CsBotRegistrationResponse {
  token: string;
  bot: CsBot;
  botSpace: CsBotSpaceBasic;
}

export interface CsBot {
  id: string;
  botSpaceId: string;
  name: string;
  capabilities: string | null;
  isManager: boolean;
  lastSeenAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface CsBotSpaceBasic {
  id: string;
  name: string;
}

export interface CsBotStatus {
  id: string;
  botSpaceId: string;
  botId: string;
  botName: string;
  status: string;
  updatedByBotId: string;
  createdAt: string;
  updatedAt: string;
}

/** Account configuration stored in the OpenClaw config file. */
export interface ClawSwarmAccountConfig {
  enabled?: boolean;
  apiUrl?: string;
  joinCode?: string;
  botName?: string;
  capabilities?: string;
  /** Pre-registered JWT — when set, registration is skipped. */
  token?: string;
  /** Whether this bot has manager privileges (used with pre-registered token). */
  isManager?: boolean;
  /** HTTP poll interval in milliseconds (default: 5000). */
  pollIntervalMs?: number;
  /** Resolved after registration. */
  botSpaceId?: string;
  /** Resolved after registration. */
  botId?: string;
}
