"use client";

import type { Bot, BotStatus } from "@/lib/types";

interface Props {
  bots: Bot[];
  statuses: BotStatus[];
}

function relativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const seconds = Math.floor((now - then) / 1000);

  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function BotStatusPanel({ bots, statuses }: Props) {
  const statusMap = new Map(statuses.map((s) => [s.botId, s]));
  const botMap = new Map(bots.map((b) => [b.id, b]));

  const withStatus = statuses
    .filter((s) => botMap.has(s.botId))
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));

  const withoutStatus = bots.filter((b) => !statusMap.has(b.id));

  return (
    <div>
      <h3 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider px-3 mb-2">
        Bot Statuses ({withStatus.length})
      </h3>
      {withStatus.length === 0 && withoutStatus.length === 0 ? (
        <p className="text-xs text-zinc-500 px-3">No bots in this space</p>
      ) : (
        <div className="space-y-2 px-3">
          {withStatus.map((s) => (
            <div key={s.botId}>
              <div className="flex items-baseline justify-between gap-2">
                <span className="text-sm font-medium text-zinc-200">{s.botName}</span>
                <span className="text-[10px] text-zinc-500 shrink-0">{relativeTime(s.updatedAt)}</span>
              </div>
              <p className="text-xs text-zinc-400 mt-0.5 whitespace-pre-wrap break-words">{s.status}</p>
            </div>
          ))}
          {withoutStatus.map((bot) => (
            <div key={bot.id}>
              <span className="text-sm font-medium text-zinc-200">{bot.name}</span>
              <p className="text-xs text-zinc-500 mt-0.5">No status</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
