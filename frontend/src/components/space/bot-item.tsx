"use client";

import type { Bot, BotStatus } from "@/lib/types";
import * as api from "@/lib/api";
import { relativeTime } from "@/lib/relative-time";

interface Props {
  bot: Bot;
  status?: BotStatus;
  isOwner: boolean;
  spaceId: string;
  onUpdated: () => void;
}

export function BotItem({ bot, status, isOwner, spaceId, onUpdated }: Props) {
  async function handleToggleManager() {
    try {
      if (bot.isManager) {
        await api.removeManager(spaceId, bot.id);
      } else {
        await api.assignManager(spaceId, bot.id);
      }
      onUpdated();
    } catch {}
  }

  async function handleRemove() {
    try {
      await api.removeBot(spaceId, bot.id);
      onUpdated();
    } catch {}
  }

  return (
    <div className="flex items-start justify-between gap-2 py-2 px-3 rounded hover:bg-zinc-800/50">
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-zinc-200 truncate">{bot.name}</span>
          {bot.isManager && (
            <span className="text-[10px] bg-amber-900/50 text-amber-400 rounded px-1.5 py-px">
              manager
            </span>
          )}
        </div>
        <span className="text-[10px] text-zinc-500">
          {bot.lastSeenAt ? `seen ${relativeTime(bot.lastSeenAt)}` : "never seen"}
        </span>
      </div>
      {isOwner && (
        <div className="flex gap-1 shrink-0">
          <button
            onClick={handleToggleManager}
            className="text-[10px] text-zinc-500 hover:text-zinc-300 px-1.5 py-0.5 rounded hover:bg-zinc-700 transition-colors"
            title={bot.isManager ? "Remove manager" : "Make manager"}
          >
            {bot.isManager ? "Demote" : "Promote"}
          </button>
          <button
            onClick={handleRemove}
            className="text-[10px] text-red-500 hover:text-red-400 px-1.5 py-0.5 rounded hover:bg-zinc-700 transition-colors"
          >
            Remove
          </button>
        </div>
      )}
    </div>
  );
}
