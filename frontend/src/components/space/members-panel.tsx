"use client";

import type { SpaceMemberWithUser } from "@/lib/types";
import * as api from "@/lib/api";

interface Props {
  members: SpaceMemberWithUser[];
  isOwner: boolean;
  spaceId: string;
  currentUserId: string;
  onUpdated: () => void;
}

export function MembersPanel({ members, isOwner, spaceId, currentUserId, onUpdated }: Props) {
  async function handleRemove(userId: string) {
    try {
      await api.removeMember(spaceId, userId);
      onUpdated();
    } catch {}
  }

  return (
    <div>
      <h3 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider px-3 mb-2">
        Members ({members.length})
      </h3>
      <div className="space-y-0.5">
        {members.map((m) => (
          <div key={m.id} className="flex items-center justify-between py-1.5 px-3 rounded hover:bg-zinc-800/50">
            <div className="flex items-center gap-2 min-w-0">
              <span className="text-sm text-zinc-200 truncate">
                {m.displayName || m.email}
              </span>
              {m.role === "owner" && (
                <span className="text-[10px] bg-blue-900/50 text-blue-400 rounded px-1.5 py-px shrink-0">
                  owner
                </span>
              )}
            </div>
            {isOwner && m.userId !== currentUserId && (
              <button
                onClick={() => handleRemove(m.userId)}
                className="text-[10px] text-red-500 hover:text-red-400 px-1.5 py-0.5 rounded hover:bg-zinc-700 transition-colors shrink-0"
              >
                Remove
              </button>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
