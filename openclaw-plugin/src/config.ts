import type { ClawSwarmAccountConfig } from "./types.js";

const DEFAULT_API_URL = "http://localhost:8080/api/v1";

interface OpenClawConfig {
  channels?: {
    "claw-swarm"?: {
      accounts?: Record<string, ClawSwarmAccountConfig>;
    };
  };
}

/** Return all account IDs under channels["claw-swarm"].accounts. */
export function listAccountIds(cfg: OpenClawConfig): string[] {
  const accounts = cfg.channels?.["claw-swarm"]?.accounts;
  return accounts ? Object.keys(accounts) : [];
}

/** Resolve a single account config, applying defaults. */
export function resolveAccount(
  cfg: OpenClawConfig,
  accountId: string,
): ClawSwarmAccountConfig {
  const raw = cfg.channels?.["claw-swarm"]?.accounts?.[accountId];
  if (!raw) {
    throw new Error(`claw-swarm account "${accountId}" not found in config`);
  }
  return {
    ...raw,
    apiUrl: raw.apiUrl || DEFAULT_API_URL,
  };
}
