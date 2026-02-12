// Entity types matching backend models

export interface User {
  id: string;
  email: string;
  displayName: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface BotSpace {
  id: string;
  ownerId: string;
  name: string;
  description: string | null;
  joinCode: string;
  managerJoinCode: string;
  managerBotId: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface SpaceMember {
  id: string;
  botSpaceId: string;
  userId: string;
  role: string;
  joinedAt: string;
}

export interface SpaceMemberWithUser extends SpaceMember {
  email: string;
  displayName: string | null;
}

export interface Bot {
  id: string;
  botSpaceId: string;
  name: string;
  capabilities: string | null;
  isManager: boolean;
  lastSeenAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface Message {
  id: string;
  botSpaceId: string;
  senderId: string;
  senderName: string;
  senderType: string;
  content: string;
  createdAt: string;
}

export interface BotStatus {
  id: string;
  botSpaceId: string;
  botId: string;
  botName: string;
  status: string;
  updatedByBotId: string;
  createdAt: string;
  updatedAt: string;
}

export interface BotSkill {
  id: string;
  botSpaceId: string;
  botId: string;
  botName: string;
  name: string;
  description: string;
  tags: string[] | null;
  createdAt: string;
  updatedAt: string;
}

export interface Summary {
  id: string;
  botSpaceId: string;
  content: string;
  createdByBotId: string;
  createdAt: string;
  updatedAt: string;
}

export interface InviteCode {
  id: string;
  botSpaceId: string;
  code: string;
  createdAt: string;
  expiresAt: string | null;
}

// Request types

export interface SignupRequest {
  email: string;
  password: string;
  displayName?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface CreateBotSpaceRequest {
  name: string;
  description?: string;
}

export interface UpdateBotSpaceRequest {
  name?: string;
  description?: string;
}

export interface PostMessageRequest {
  content: string;
}

export interface JoinBotSpaceRequest {
  inviteCode: string;
}

// Response types

export interface AuthResponse {
  token: string;
  user: User;
}

export interface MessageListResponse {
  messages: Message[];
  count: number;
  hasMore: boolean;
}

export interface OverallResponse {
  messages: MessageListResponse;
  summary: Summary | null;
}
