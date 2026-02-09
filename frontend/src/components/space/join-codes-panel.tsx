"use client";

import { useState } from "react";
import type { BotSpace } from "@/lib/types";
import * as api from "@/lib/api";

interface Props {
  space: BotSpace;
  onUpdated: (space: BotSpace) => void;
}

export function JoinCodesPanel({ space, onUpdated }: Props) {
  const [regenerating, setRegenerating] = useState(false);

  async function handleCopy(text: string) {
    try {
      await navigator.clipboard.writeText(text);
    } catch {}
  }

  async function handleRegenerate() {
    setRegenerating(true);
    try {
      const updated = await api.regenerateJoinCodes(space.id);
      onUpdated(updated);
    } catch {}
    setRegenerating(false);
  }

  return (
    <div>
      <h3 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider px-3 mb-2">
        Join Codes
      </h3>
      <div className="px-3 space-y-3">
        <div>
          <label className="text-xs text-zinc-500">Bot Join Code</label>
          <div className="flex items-center gap-2 mt-0.5">
            <code className="flex-1 text-xs bg-zinc-800 rounded px-2 py-1 text-zinc-300 truncate">
              {space.joinCode}
            </code>
            <button
              onClick={() => handleCopy(space.joinCode)}
              className="text-xs text-blue-400 hover:text-blue-300 shrink-0"
            >
              Copy
            </button>
          </div>
        </div>
        <div>
          <label className="text-xs text-zinc-500">Manager Join Code</label>
          <div className="flex items-center gap-2 mt-0.5">
            <code className="flex-1 text-xs bg-zinc-800 rounded px-2 py-1 text-zinc-300 truncate">
              {space.managerJoinCode}
            </code>
            <button
              onClick={() => handleCopy(space.managerJoinCode)}
              className="text-xs text-blue-400 hover:text-blue-300 shrink-0"
            >
              Copy
            </button>
          </div>
        </div>
        <button
          onClick={handleRegenerate}
          disabled={regenerating}
          className="text-xs text-zinc-400 hover:text-zinc-200 disabled:opacity-50 transition-colors"
        >
          {regenerating ? "Regenerating..." : "Regenerate codes"}
        </button>
      </div>
    </div>
  );
}
