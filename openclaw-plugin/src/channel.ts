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

async function buildManagerContext(
  client: ClawSwarmClient,
  botSpaceId: string,
  msg: CsMessage,
  log?: (msg: string) => void,
): Promise<string> {
  const [statuses, inProgressTasks, availableTasks] = await Promise.all([
    client.listStatuses(botSpaceId),
    client.listTasks(botSpaceId, { status: "in_progress" }),
    client.listTasks(botSpaceId, { status: "available" }),
  ]);
  log?.(`fetched manager context: ${statuses.length} bot statuses, ${inProgressTasks.length} in-progress tasks, ${availableTasks.length} available tasks`);

  let ctx = "\n<manager-context>";

  ctx += "\n<bot-statuses>";
  for (const s of statuses) {
    ctx += `\n<bot name="${s.botName}" id="${s.botId}" status="${s.status}" />`;
  }
  ctx += "\n</bot-statuses>";

  ctx += "\n<tasks-in-progress>";
  for (const t of inProgressTasks) {
    ctx += `\n<task name="${t.name}" id="${t.id}" botId="${t.botId ?? ""}">${t.description}</task>`;
  }
  ctx += "\n</tasks-in-progress>";

  ctx += "\n<tasks-available>";
  for (const t of availableTasks) {
    ctx += `\n<task name="${t.name}" id="${t.id}">${t.description}</task>`;
  }
  ctx += "\n</tasks-available>";

  ctx += "\n</manager-context>";
  if (msg.senderType === "bot") {
    ctx += "\n<manager-instructions>";
    ctx += "\nIf you need to update bot statuses or manage tasks, include a <manager-actions> block in your reply.";
    ctx += "\nAvailable actions:";
    ctx += '\n  <status-update botId="BOT_ID" status="new status text" />';
    ctx += '\n  <task-create name="task name" description="task description" />';
    ctx += '\n  <task-create name="task name" description="task description" botId="BOT_ID" />';
    ctx += '\n  <task-assign taskId="TASK_ID" botId="BOT_ID" />';
    ctx += "\nThe <manager-actions> block will be stripped from your message before sending.";
    ctx += "\n</manager-instructions>";
  }
  return ctx;
}

async function parseAndExecuteManagerActions(
  client: ClawSwarmClient,
  botSpaceId: string,
  text: string,
  log?: (msg: string) => void,
): Promise<string> {
  log?.(`parsing manager actions... \n${text}`);
  const actionBlockRegex = /<manager-actions>([\s\S]*?)<\/manager-actions>/g;
  const matches = [...text.matchAll(actionBlockRegex)];
  if (matches.length === 0) return text;

  for (const match of matches) {
    const block = match[1];

    const statusUpdates = [
      ...block.matchAll(
        /<status-update\s+botId="([^"]+)"\s+status="([^"]+)"\s*\/>/g,
      ),
    ];
    for (const [, botId, status] of statusUpdates) {
      try {
        log?.(`updating status for bot ${botId} to "${status}"...`);
        await client.updateBotStatus(botSpaceId, botId, status);
        log?.(`manager action: updated status for ${botId} to "${status}"`);
      } catch (err) {
        log?.(`manager action failed (status-update): ${err}`);
      }
    }

    const taskCreates = [
      ...block.matchAll(
        /<task-create\s+name="([^"]+)"\s+description="([^"]+)"(?:\s+botId="([^"]+)")?\s*\/>/g,
      ),
    ];
    for (const [, name, description, botId] of taskCreates) {
      try {
        log?.(`creating task "${name}"${botId ? ` assigned to ${botId}` : ""}...`);
        await client.createTask(
          botSpaceId,
          name,
          description,
          botId || undefined,
        );
        log?.(
          `manager action: created task "${name}"${botId ? ` assigned to ${botId}` : ""}`,
        );
      } catch (err) {
        log?.(`manager action failed (task-create): ${err}`);
      }
    }

    const taskAssigns = [
      ...block.matchAll(
        /<task-assign\s+taskId="([^"]+)"\s+botId="([^"]+)"\s*\/>/g,
      ),
    ];
    for (const [, taskId, botId] of taskAssigns) {
      try {
        await client.assignTask(botSpaceId, taskId, botId);
        log?.(`manager action: assigned task ${taskId} to ${botId}`);
      } catch (err) {
        log?.(`manager action failed (task-assign): ${err}`);
      }
    }
  }

  return text.replace(actionBlockRegex, "").trim();
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

              let body = history
                .map(
                  (m) =>
                    `<message sender="${m.senderName}" id="${m.id}">\n${m.content}\n</message>`,
                )
                .join("\n");

              if (acct.isManager) {
                log?.(`fetching manager context for message ${msg.id}...`);
                try {
                  body = await buildManagerContext(client, acct.botSpaceId, msg, log) + "\n" + body;
                 
                } catch (err) {
                  log?.(`failed to fetch manager context: ${err}`);
                }
              }

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
                  deliver: async (payload: { text?: string, isError?: boolean}) => {
                    log?.(`delivering reply to message ${msg.id}... ${payload.text} ${payload.isError ? "(marked as error)" : ""}`);
                    if (!payload.text) return;
                    let text = payload.text;
                    
                    if (acct.isManager && msg.senderType === "bot") {
                      log?.(`processing manager actions in reply...`);
                      text = await parseAndExecuteManagerActions(
                        client,
                        acct.botSpaceId,
                        text,
                        log,
                      );
                    }
                    if (text) {
                      await client.sendMessage(acct.botSpaceId!, text);
                    }
                  },
                  onSkip: (payload:{text?: string}, info: { kind: string; reason: string }) => {
                    log?.(`reply skipped: ${info.kind} - ${info.reason}: ${payload.text ?? "[empty]"}`);
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
