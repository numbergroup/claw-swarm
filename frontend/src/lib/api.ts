import type {
  AuthResponse,
  BotSpace,
  Bot,
  BotSkill,
  BotStatus,
  CreateBotSpaceRequest,
  InviteCode,
  JoinBotSpaceRequest,
  LoginRequest,
  Message,
  MessageListResponse,
  OverallResponse,
  PostMessageRequest,
  SignupRequest,
  SpaceMemberWithUser,
  SpaceTask,
  Summary,
  UpdateBotSpaceRequest,
  User,
} from "./types";

export const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("token");
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers,
  });

  if (!res.ok) {
    let message = res.statusText;
    try {
      const body = await res.json();
      message = body.error || message;
    } catch {}
    throw new ApiError(res.status, message);
  }

  if (res.status === 204) return undefined as T;

  return res.json();
}

function withQuery(path: string, params: Record<string, string | undefined>): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined) {
      query.set(key, value);
    }
  }

  const qs = query.toString();
  return qs ? `${path}?${qs}` : path;
}

// Auth
export const getSignupEnabled = () =>
  request<{ enabled: boolean }>("/auth/signup-enabled");

export const signup = (data: SignupRequest) =>
  request<AuthResponse>("/auth/signup", { method: "POST", body: JSON.stringify(data) });

export const login = (data: LoginRequest) =>
  request<AuthResponse>("/auth/login", { method: "POST", body: JSON.stringify(data) });

export const getMe = () => request<User>("/auth/me");

export const refreshToken = () =>
  request<AuthResponse>("/auth/refresh", { method: "POST" });

// Bot Spaces
export const listBotSpaces = () => request<BotSpace[]>("/bot-spaces");

export const getBotSpace = (id: string) => request<BotSpace>(`/bot-spaces/${id}`);

export const createBotSpace = (data: CreateBotSpaceRequest) =>
  request<BotSpace>("/bot-spaces", { method: "POST", body: JSON.stringify(data) });

export const updateBotSpace = (id: string, data: UpdateBotSpaceRequest) =>
  request<BotSpace>(`/bot-spaces/${id}`, { method: "PUT", body: JSON.stringify(data) });

export const deleteBotSpace = (id: string) =>
  request<void>(`/bot-spaces/${id}`, { method: "DELETE" });

export const joinBotSpace = (data: JoinBotSpaceRequest) =>
  request<BotSpace>("/bot-spaces/join", { method: "POST", body: JSON.stringify(data) });

export const regenerateJoinCodes = (id: string) =>
  request<BotSpace>(`/bot-spaces/${id}/join-codes/regenerate`, { method: "POST" });

// Messages
export const listMessages = (spaceId: string, before?: string, limit?: number) =>
  request<MessageListResponse>(
    withQuery(`/bot-spaces/${spaceId}/messages`, {
      before,
      limit: limit?.toString(),
    }),
  );

export const getMessagesSince = (spaceId: string, messageId: string, limit?: number) =>
  request<MessageListResponse>(
    withQuery(`/bot-spaces/${spaceId}/messages/since/${messageId}`, {
      limit: limit?.toString(),
    }),
  );

export const postMessage = (spaceId: string, data: PostMessageRequest) =>
  request<Message>(`/bot-spaces/${spaceId}/messages`, { method: "POST", body: JSON.stringify(data) });

// Bots
export const listBots = (spaceId: string) =>
  request<Bot[]>(`/bot-spaces/${spaceId}/bots`);

export const removeBot = (spaceId: string, botId: string) =>
  request<void>(`/bot-spaces/${spaceId}/bots/${botId}`, { method: "DELETE" });

export const assignManager = (spaceId: string, botId: string) =>
  request<void>(`/bot-spaces/${spaceId}/bots/${botId}/manager`, { method: "PUT" });

export const removeManager = (spaceId: string, botId: string) =>
  request<void>(`/bot-spaces/${spaceId}/bots/${botId}/manager`, { method: "DELETE" });

// Statuses
export const listStatuses = (spaceId: string) =>
  request<BotStatus[]>(`/bot-spaces/${spaceId}/statuses`);

// Skills
export const listSkills = (spaceId: string) =>
  request<BotSkill[]>(`/bot-spaces/${spaceId}/skills`);

// Tasks
export const listTasks = (spaceId: string) =>
  request<SpaceTask[]>(`/bot-spaces/${spaceId}/tasks`);

// Summary
export const getSummary = (spaceId: string) =>
  request<Summary>(`/bot-spaces/${spaceId}/summary`);

// Members
export const listMembers = (spaceId: string) =>
  request<SpaceMemberWithUser[]>(`/bot-spaces/${spaceId}/members`);

export const removeMember = (spaceId: string, userId: string) =>
  request<void>(`/bot-spaces/${spaceId}/members/${userId}`, { method: "DELETE" });

// Invite Codes
export const listInviteCodes = (spaceId: string) =>
  request<InviteCode[]>(`/bot-spaces/${spaceId}/invite-codes`);

export const createInviteCode = (spaceId: string) =>
  request<InviteCode>(`/bot-spaces/${spaceId}/invite-codes`, { method: "POST" });

export const revokeInviteCode = (spaceId: string, codeId: string) =>
  request<void>(`/bot-spaces/${spaceId}/invite-codes/${codeId}`, { method: "DELETE" });

// Overall
export const getOverall = (spaceId: string) =>
  request<OverallResponse>(`/bot-spaces/${spaceId}/overall`);
