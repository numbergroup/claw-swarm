"use client";

import { useState, useRef, useEffect } from "react";
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
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    function handleClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [menuOpen]);

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

  async function handleToggleMute() {
    try {
      if (bot.isMuted) {
        await api.unmuteBot(spaceId, bot.id);
      } else {
        await api.muteBot(spaceId, bot.id);
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
          {bot.isMuted && (
            <span className="text-[10px] bg-zinc-700/50 text-zinc-400 rounded px-1.5 py-px">
              muted
            </span>
          )}
        </div>
        <span className="text-[10px] text-zinc-500">
          {bot.lastSeenAt ? `seen ${relativeTime(bot.lastSeenAt)}` : "never seen"}
        </span>
      </div>
      {isOwner && (
        <div className="relative shrink-0" ref={menuRef}>
          <button
            onClick={() => setMenuOpen((v) => !v)}
            className="p-1 rounded hover:bg-zinc-700 transition-colors text-zinc-500 hover:text-zinc-300"
          >
            <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor">
              <rect x="2" y="3" width="12" height="1.5" rx="0.75" />
              <rect x="2" y="7.25" width="12" height="1.5" rx="0.75" />
              <rect x="2" y="11.5" width="12" height="1.5" rx="0.75" />
            </svg>
          </button>
          {menuOpen && (
            <div className="absolute right-0 top-full mt-1 z-50 min-w-[120px] bg-zinc-800 border border-zinc-700 rounded-md shadow-lg py-1">
              <button
                onClick={() => { handleToggleManager(); setMenuOpen(false); }}
                className="w-full text-left text-xs text-zinc-300 hover:bg-zinc-700 px-3 py-1.5 transition-colors"
              >
                {bot.isManager ? "Demote" : "Promote"}
              </button>
              <button
                onClick={() => { handleToggleMute(); setMenuOpen(false); }}
                className="w-full text-left text-xs text-zinc-300 hover:bg-zinc-700 px-3 py-1.5 transition-colors"
              >
                {bot.isMuted ? "Unmute" : "Mute"}
              </button>
              <button
                onClick={() => { handleRemove(); setMenuOpen(false); }}
                className="w-full text-left text-xs text-red-400 hover:bg-zinc-700 px-3 py-1.5 transition-colors"
              >
                Remove
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
