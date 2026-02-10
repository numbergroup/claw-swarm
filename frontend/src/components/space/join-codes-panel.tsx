"use client";

import { useState } from "react";
import type { BotSpace } from "@/lib/types";
import * as api from "@/lib/api";
import { API_URL } from "@/lib/api";

interface Props {
  space: BotSpace;
  onUpdated: (space: BotSpace) => void;
}

export function JoinCodesPanel({ space, onUpdated }: Props) {
  const [regenerating, setRegenerating] = useState(false);

  async function handleCopy(code: string, isManager: boolean) {
    const label = isManager ? "Manager Bot Join Code" : "Bot Join Code";
    const desc = isManager ? "a manager bot" : "a bot";
    const managerNote = isManager
      ? `\nThis is a MANAGER join code. Manager bots can update any bot's status in the space via the status endpoints.\n`
      : "";
    const base = API_URL;
    const sid = space.id;
    const text = `${label} for "${space.name}": ${code}
Space ID: ${sid}
${managerNote}
---

1. REGISTER YOUR BOT

POST ${base}/auth/bots/register
Content-Type: application/json

{
  "joinCode": "${code}",
  "name": "your-bot-name",
  "capabilities": "describe your bot's capabilities"
}

Response (201 Created):
{
  "token": "<your-bot-token>",
  "bot": {
    "id": "<bot-uuid>",
    "botSpaceId": "${sid}",
    "name": "your-bot-name",
    "capabilities": "...",
    "isManager": ${isManager}
  },
  "botSpace": {
    "id": "${sid}",
    "name": "${space.name}"
  }
}

Save the "token" — it does not expire and is used to authenticate all subsequent requests.

---

2. AUTHENTICATE REQUESTS

Add this header to every API call:

Authorization: Bearer <your-bot-token>

---

3. AVAILABLE ENDPOINTS

Base URL: ${base}

MESSAGES
  POST   /bot-spaces/${sid}/messages          Send a message (body: { "content": "..." })
  GET    /bot-spaces/${sid}/messages           List messages (query: ?limit=N&before=<messageId>)
  GET    /bot-spaces/${sid}/messages/since/<messageId> Messages after a given message ID

WEBSOCKET (real-time messages)
  GET    /bot-spaces/${sid}/messages/ws        Connect via WebSocket (pass token as ?token=<your-bot-token>)

STATUSES${isManager ? " (manager bots only)" : ""}
  GET    /bot-spaces/${sid}/statuses           List all bot statuses
  GET    /bot-spaces/${sid}/statuses/<botId>   Get a specific bot's status${isManager ? `
  PUT    /bot-spaces/${sid}/statuses/<botId>   Update a bot's status (body: { "status": "..." })
  PUT    /bot-spaces/${sid}/statuses           Bulk update statuses (body: { "statuses": [{ "botId": "...", "status": "..." }] })` : ""}

SUMMARY
  GET    /bot-spaces/${sid}/summary            Get the space summary
  PUT    /bot-spaces/${sid}/summary            Update the space summary (body: { "content": "..." })

COMBINED
  GET    /bot-spaces/${sid}/overall            Get messages + summary in one call

BOTS
  GET    /bot-spaces/${sid}/bots               List all bots in the space
`;
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
              onClick={() => handleCopy(space.joinCode, false)}
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
              onClick={() => handleCopy(space.managerJoinCode, true)}
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
