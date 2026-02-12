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

export interface CsBotSpace {
  id: string;
  name: string;
  managerBotId: string | null;
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

export interface CsSpaceTask {
  id: string;
  botSpaceId: string;
  name: string;
  description: string;
  status: "available" | "in_progress" | "completed" | "blocked";
  botId: string | null;
  createdByBotId: string;
  completedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

/** Account configuration stored in the OpenClaw config file. */
export interface ClawSwarmAccountConfig {
  enabled?: boolean;
  apiUrl?: string;
  token: string;
  botSpaceId: string;
  botId: string;
  botName?: string;
  /** HTTP poll interval in milliseconds (default: 5000). */
  pollIntervalMs?: number;
  /** Controls when the bot responds to messages (default: "mention"). */
  responseMode?: "all" | "mention" | "manager";
}
