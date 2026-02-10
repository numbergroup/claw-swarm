"use client";

import { useState } from "react";
import type { InviteCode } from "@/lib/types";
import * as api from "@/lib/api";

interface Props {
  spaceId: string;
  codes: InviteCode[];
  onUpdated: () => void;
}

export function InviteCodesPanel({ spaceId, codes, onUpdated }: Props) {
  const [generating, setGenerating] = useState(false);

  async function handleCopy(code: string) {
    try {
      const url = `${window.location.origin}/?invite=${code}`;
      await navigator.clipboard.writeText(url);
    } catch {}
  }

  async function handleGenerate() {
    setGenerating(true);
    try {
      await api.createInviteCode(spaceId);
      onUpdated();
    } catch {}
    setGenerating(false);
  }

  async function handleRevoke(codeId: string) {
    try {
      await api.revokeInviteCode(spaceId, codeId);
      onUpdated();
    } catch {}
  }

  return (
    <div>
      <div className="flex items-center justify-between px-3 mb-2">
        <h3 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
          Invite Codes
        </h3>
        <button
          onClick={handleGenerate}
          disabled={generating}
          className="text-xs text-blue-400 hover:text-blue-300 disabled:opacity-50"
        >
          {generating ? "..." : "+ New"}
        </button>
      </div>
      <div className="px-3 space-y-2">
        {codes.length === 0 ? (
          <p className="text-xs text-zinc-500">No invite codes</p>
        ) : (
          codes.map((code) => (
            <div key={code.id} className="flex items-center gap-2">
              <code className="flex-1 text-xs bg-zinc-800 rounded px-2 py-1 text-zinc-300 truncate">
                {code.code}
              </code>
              <button
                onClick={() => handleCopy(code.code)}
                className="text-xs text-blue-400 hover:text-blue-300 shrink-0"
              >
                Copy
              </button>
              <button
                onClick={() => handleRevoke(code.id)}
                className="text-xs text-red-500 hover:text-red-400 shrink-0"
              >
                Revoke
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
