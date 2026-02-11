import { ClawSwarmClient } from "./client.js";
import { listAccountIds, resolveAccount } from "./config.js";
import type { CsMessage } from "./types.js";

interface AccountState {
  client: ClawSwarmClient;
  botSpaceId: string;
  botId: string;
  isManager: boolean;
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

        let isManager: boolean;

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
          isManager = acct.isManager ?? false;
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
          isManager = reg.bot.isManager;
        }

        const state: AccountState = { client, botSpaceId, botId, isManager };
        accounts.set(accountId, state);

        client.connectWebSocket(botSpaceId, (msg: CsMessage) => {
          // Filter out our own messages to prevent echo loops
          if (msg.senderId === botId) return;

          if (state.isManager && msg.senderType === "bot") {
            generateAndUpdateStatus(state, msg, api).catch(() => {});
          }

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

async function generateAndUpdateStatus(
  state: AccountState,
  msg: CsMessage,
  api: OpenClawApi,
): Promise<void> {
  const { messages } = await state.client.getMessages(state.botSpaceId, {
    limit: 20,
  });

  const transcript = messages
    .map((m) => `[${m.senderName}]: ${m.content}`)
    .join("\n");

  const systemPrompt =
    "You generate short status descriptions for bots in a chat space. " +
    "Respond with only the status text, no quotes or extra formatting. " +
    "Keep it under 100 characters.";

  const prompt =
    `Here is the recent chat history:\n\n${transcript}\n\n` +
    `Based on the above chat history, write a brief status for the bot named "${msg.senderName}" ` +
    `that reflects what they are currently doing or talking about.`;

  const generatedStatus: string = await api.generateText({
    prompt,
    systemPrompt,
  });

  await state.client.updateBotStatus(
    state.botSpaceId,
    msg.senderId,
    generatedStatus,
  );
}
