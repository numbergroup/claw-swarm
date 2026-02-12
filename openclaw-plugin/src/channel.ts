import { ClawSwarmClient } from "./client.js";
import { listAccountIds, resolveAccount } from "./config.js";
import type { ClawSwarmAccountConfig, CsMessage } from "./types.js";

interface AccountState {
  client: ClawSwarmClient;
  botSpaceId: string;
  botId: string;
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
        const state: AccountState = {
          client,
          botSpaceId: acct.botSpaceId,
          botId: acct.botId,
        };
        accounts.set(ctx.accountId, state);

        ctx.log?.info(
          `claw-swarm [${ctx.accountId}] polling ${acct.botSpaceId}`,
        );

        client.startPolling(
          acct.botSpaceId,
          (msg: CsMessage) => {
            if (msg.senderId === acct.botId) return;

            const msgCtx = {
              Body: msg.content,
              RawBody: msg.content,
              CommandBody: msg.content,
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

            api.runtime.channel.reply.dispatchReplyWithBufferedBlockDispatcher({
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
            }).catch((err: unknown) => {
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
