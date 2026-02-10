import { ClawSwarmClient } from "./client.js";
import { listAccountIds, resolveAccount } from "./config.js";
import type { CsMessage } from "./types.js";

interface AccountState {
  client: ClawSwarmClient;
  botSpaceId: string;
  botId: string;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type OpenClawApi = any;

export function createChannel(api: OpenClawApi) {
  const accounts = new Map<string, AccountState>();

  return {
    id: "claw-swarm",

    meta: {
      label: "Claw-Swarm",
      aliases: ["clawswarm", "botspace"],
    },

    capabilities: {
      chatTypes: ["group"],
    },

    config: {
      listAccountIds,
      resolveAccount,
    },

    outbound: {
      async sendText({
        text,
        accountId,
      }: {
        text: string;
        accountId: string;
      }): Promise<void> {
        const state = accounts.get(accountId);
        if (!state) {
          throw new Error(
            `claw-swarm account "${accountId}" is not connected`,
          );
        }
        await state.client.sendMessage(state.botSpaceId, text);
      },
    },

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    async setup(cfg: any): Promise<void> {
      const ids = listAccountIds(cfg);

      for (const accountId of ids) {
        const acct = resolveAccount(cfg, accountId);
        if (acct.enabled === false) continue;

        const client = new ClawSwarmClient(acct.apiUrl!);

        let botSpaceId: string;
        let botId: string;

        if (acct.token) {
          // Use pre-registered token
          client.setToken(acct.token);
          if (!acct.botSpaceId || !acct.botId) {
            throw new Error(
              `claw-swarm account "${accountId}": when using a pre-registered token, botSpaceId and botId are required`,
            );
          }
          botSpaceId = acct.botSpaceId;
          botId = acct.botId;
        } else {
          // Register via join code
          if (!acct.joinCode) {
            throw new Error(
              `claw-swarm account "${accountId}": either joinCode or token is required`,
            );
          }
          const reg = await client.register(
            acct.joinCode,
            acct.botName || "openclaw-agent",
            acct.capabilities || "AI assistant",
          );
          botSpaceId = reg.botSpace.id;
          botId = reg.bot.id;
        }

        const state: AccountState = { client, botSpaceId, botId };
        accounts.set(accountId, state);

        client.connectWebSocket(botSpaceId, (msg: CsMessage) => {
          // Filter out our own messages to prevent echo loops
          if (msg.senderId === botId) return;

          api.dispatchMessage({
            channel: "claw-swarm",
            scope: "group",
            peer: msg.botSpaceId,
            accountId,
            senderId: msg.senderId,
            senderName: msg.senderName,
            text: msg.content,
          });
        });
      }
    },

    async teardown(): Promise<void> {
      for (const [, state] of accounts) {
        state.client.disconnectWebSocket();
      }
      accounts.clear();
    },
  };
}
