import { ClawSwarmClient } from "./client.js";
import { listAccountIds, resolveAccount } from "./config.js";
import type { ClawSwarmAccountConfig, CsMessage } from "./types.js";

interface AccountState {
  client: ClawSwarmClient;
  botSpaceId: string;
  botId: string;
  managerBotId?: string;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type OpenClawApi = any;

function resolveState(
  accounts: Map<string, AccountState>,
  accountId?: string | null,
  to?: string,
): AccountState {
  if (accountId) {
    const s = accounts.get(accountId);
    if (s) return s;
  }
  // Fall back to first account whose botSpaceId matches `to`
  for (const [, s] of accounts) {
    if (s.botSpaceId === to) return s;
  }
  // Fall back to the only account if there's exactly one
  if (accounts.size === 1) return accounts.values().next().value!;
  throw new Error(
    `claw-swarm: cannot resolve account for accountId=${accountId}, to=${to}`,
  );
}

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

    resolver: {
      async resolveTargets({ inputs }: { inputs: string[] }) {
        return inputs.map((input) => ({
          input,
          resolved: true,
          id: input,
        }));
      },
    },

    outbound: {
      deliveryMode: "direct" as const,

      async sendText({
        text,
        to,
        accountId,
      }: {
        text: string;
        to: string;
        accountId?: string | null;
      }) {
        const state = resolveState(accounts, accountId, to);
        await state.client.sendMessage(state.botSpaceId, text);
        return {
          channel: "claw-swarm" as const,
          messageId: crypto.randomUUID(),
        };
      },

      async sendMedia({
        text,
        mediaUrl,
        to,
        accountId,
      }: {
        text: string;
        mediaUrl?: string;
        to: string;
        accountId?: string | null;
      }) {
        const state = resolveState(accounts, accountId, to);
        await state.client.sendMessage(
          state.botSpaceId,
          text || `[Media: ${mediaUrl ?? "unknown"}]`,
        );
        return {
          channel: "claw-swarm" as const,
          messageId: crypto.randomUUID(),
        };
      },
    },

    gateway: {
      async startAccount(ctx: {
        accountId: string;
        account: ClawSwarmAccountConfig;
        cfg?: Record<string, unknown>;
        log?: { info: (msg: string) => void };
      }) {
        const acct = ctx.account;
        const log = ctx.log
          ? (msg: string) => ctx.log!.info(msg)
          : undefined;
        const client = new ClawSwarmClient(acct.apiUrl!, log);

        if (!acct.token || !acct.botSpaceId || !acct.botId) {
          throw new Error(
            `claw-swarm account "${ctx.accountId}": token, botSpaceId, and botId are required`,
          );
        }

        client.setToken(acct.token);

        const responseMode = acct.responseMode ?? "mention";
        const botName = acct.botName;

        const state: AccountState = {
          client,
          botSpaceId: acct.botSpaceId,
          botId: acct.botId,
        };

        if (responseMode === "manager") {
          try {
            const botSpace = await client.getBotSpace(acct.botSpaceId);
            if (botSpace.managerBotId) {
              state.managerBotId = botSpace.managerBotId;
            } else {
              log?.(`warning: responseMode is "manager" but bot space has no managerBotId`);
            }
          } catch (err) {
            log?.(`warning: failed to fetch bot space for manager mode: ${err}`);
          }
        }

        if ((responseMode === "mention" || responseMode === "manager") && !botName) {
          log?.(`warning: responseMode "${responseMode}" requires botName to detect mentions`);
        }

        accounts.set(ctx.accountId, state);

        client.startPolling(
          acct.botSpaceId,
          (msg: CsMessage) => {
            if (msg.senderId === acct.botId) return;

            if (responseMode !== "all") {
              const isMentioned = botName
                ? msg.content.toLowerCase().includes(botName.toLowerCase())
                : false;

              if (responseMode === "mention" && !isMentioned) return;

              if (responseMode === "manager") {
                const isFromManager = state.managerBotId
                  ? msg.senderId === state.managerBotId
                  : false;
                if (!isMentioned && !isFromManager) return;
              }
            }

            (async () => {
              let history: CsMessage[] = [];
              try {
                const resp = await client.getMessages(acct.botSpaceId, { limit: 15 });
                history = resp.messages;
              } catch (err) {
                log?.(`failed to fetch message history: ${err}`);
                history = [msg];
              }

              // getMessages returns newest-first; reverse to chronological
              history.reverse();
              // Deduplicate and ensure triggering message is last
              history = history.filter((m) => m.id !== msg.id);
              history.push(msg);

              const body = history
                .map(
                  (m) =>
                    `<message sender="${m.senderName}" id="${m.id}">\n${m.content}\n</message>`,
                )
                .join("\n");

              const msgCtx = {
                Body: body,
                RawBody: body,
                CommandBody: body,
                From: msg.senderId,
                To: acct.botSpaceId,
                SessionKey: `claw-swarm:${acct.botSpaceId}`,
                AccountId: ctx.accountId,
                ChatType: "group",
                SenderName: msg.senderName,
                SenderId: msg.senderId,
                Provider: "claw-swarm",
                Surface: "claw-swarm",
                OriginatingChannel: "claw-swarm",
                OriginatingTo: acct.botSpaceId,
                MessageSid: msg.id,
              };

              const cfg = ctx.cfg ?? api.config;

              await api.runtime.channel.reply.dispatchReplyWithBufferedBlockDispatcher({
                ctx: msgCtx,
                cfg,
                dispatcherOptions: {
                  deliver: async (payload: { text?: string }) => {
                    if (payload.text) {
                      await client.sendMessage(acct.botSpaceId!, payload.text);
                    }
                  },
                  onError: (err: unknown) => {
                    log?.(`reply delivery error: ${err}`);
                  },
                },
              });
            })().catch((err: unknown) => {
              log?.(`dispatch error: ${err}`);
            });
          },
          { intervalMs: acct.pollIntervalMs },
        );
      },

      async stopAccount(ctx: { accountId: string }) {
        const state = accounts.get(ctx.accountId);
        if (state) {
          state.client.stopPolling();
          accounts.delete(ctx.accountId);
        }
      },
    },
  };
}
