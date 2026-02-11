import { readFile, writeFile, mkdir } from "node:fs/promises";
import { homedir } from "node:os";
import { join } from "node:path";

export interface AccountCredentials {
  token: string;
  botId: string;
  botSpaceId: string;
  isManager: boolean;
}

interface StateFile {
  accounts: Record<string, AccountCredentials>;
}

const STATE_DIR = join(homedir(), ".openclaw");
const STATE_PATH = join(STATE_DIR, "claw-swarm-state.json");

export async function loadState(): Promise<StateFile> {
  try {
    const raw = await readFile(STATE_PATH, "utf-8");
    return JSON.parse(raw) as StateFile;
  } catch {
    return { accounts: {} };
  }
}

export async function saveAccountState(
  accountId: string,
  creds: AccountCredentials,
): Promise<void> {
  const state = await loadState();
  state.accounts[accountId] = creds;
  await mkdir(STATE_DIR, { recursive: true });
  await writeFile(STATE_PATH, JSON.stringify(state, null, 2) + "\n", "utf-8");
}
