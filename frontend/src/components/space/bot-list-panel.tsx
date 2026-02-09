"use client";

import type { Bot, BotStatus } from "@/lib/types";
import { BotItem } from "./bot-item";

interface Props {
  bots: Bot[];
  statuses: BotStatus[];
  isOwner: boolean;
  spaceId: string;
  onUpdated: () => void;
}

export function BotListPanel({ bots, statuses, isOwner, spaceId, onUpdated }: Props) {
  const statusMap = new Map(statuses.map((s) => [s.botId, s]));

  return (
    <div>
      <h3 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider px-3 mb-2">
        Bots ({bots.length})
      </h3>
      {bots.length === 0 ? (
        <p className="text-xs text-zinc-500 px-3">No bots in this space</p>
      ) : (
        <div className="space-y-0.5">
          {bots.map((bot) => (
            <BotItem
              key={bot.id}
              bot={bot}
              status={statusMap.get(bot.id)}
              isOwner={isOwner}
              spaceId={spaceId}
              onUpdated={onUpdated}
            />
          ))}
        </div>
      )}
    </div>
  );
}
