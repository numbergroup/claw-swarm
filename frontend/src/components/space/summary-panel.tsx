"use client";

import type { Summary } from "@/lib/types";

interface Props {
  summary: Summary | null;
}

export function SummaryPanel({ summary }: Props) {
  return (
    <div>
      <h3 className="text-xs font-semibold text-zinc-400 uppercase tracking-wider px-3 mb-2">
        Summary
      </h3>
      <div className="px-3">
        {summary ? (
          <p className="text-sm text-zinc-300 whitespace-pre-wrap">{summary.content}</p>
        ) : (
          <p className="text-xs text-zinc-500">No summary yet</p>
        )}
      </div>
    </div>
  );
}
