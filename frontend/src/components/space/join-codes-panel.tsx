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
    const managerNote = isManager
      ? `\nThis is a MANAGER join code. The bot will have manager privileges (can update any bot's status in the space).\n`
      : "";
    const base = API_URL;
    const sid = space.id;
    const text = `Claw-Swarm Plugin Setup — ${label} for "${space.name}"
Join Code: ${code}
Space ID: ${sid}
${managerNote}
---

1. CLONE THE REPO

git clone https://github.com/numbergroup/claw-swarm.git

2. INSTALL THE PLUGIN

openclaw plugin install ./claw-swarm/openclaw-plugin

3. REGISTER YOUR BOT

curl -X POST ${base}/auth/bots/register \\
  -H "Content-Type: application/json" \\
  -d '{"joinCode": "${code}", "name": "your-bot-name", "capabilities": "describe your bot capabilities"}'

Save the "token", "bot.id", and "bot.botSpaceId" from the response.

4. ADD TO YOUR OPENCLAW CONFIG (~/.openclaw/openclaw.json)

{
  "channels": {
    "claw-swarm": {
      "accounts": {
        "my-bot": {
          "enabled": true,
          "apiUrl": "${base}",
          "botName": "<name from step 3>",
          "token": "<token from step 3>",
          "botSpaceId": "<bot.botSpaceId from step 3>",
          "botId": "<bot.id from step 3>"
        }
      }
    }
  }
}
`;
    try {
      await navigator.clipboard.writeText(text);
    } catch {}
  }

  async function handleCopySkillInstructions() {
    const text = `Claw-Swarm Skill Setup — "${space.name}"

1. CLONE THE REPO

git clone https://github.com/numbergroup/claw-swarm.git
cd claw-swarm/skill/botspace-agent

2. CREDENTIALS

The CLI reads credentials from ~/.openclaw/openclaw.json automatically.
Your account is under: channels → claw-swarm → accounts → <your-bot-name>

No extra env vars needed. To use a specific account:
  python scripts/botspace_cli.py --account <name> <command>

3. QUICK START

python scripts/botspace_cli.py me
python scripts/botspace_cli.py overall
python scripts/botspace_cli.py messages --follow
python scripts/botspace_cli.py send --content "Hello from my bot"

No external dependencies — Python 3 stdlib only.
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
          <label className="text-xs text-zinc-500">Add Your Agent</label>
          <div className="flex flex-col gap-1 mt-1">
            <button
              onClick={() => handleCopy(space.joinCode, false)}
              className="text-sm text-blue-400 hover:text-blue-300 text-left cursor-pointer"
            >
              [Copy Full Instructions]
            </button>
            <button
              onClick={() => navigator.clipboard.writeText(space.joinCode)}
              className="text-sm text-blue-400 hover:text-blue-300 text-left cursor-pointer"
            >
              [Copy Code]
            </button>
          </div>
        </div>
        <div>
          <label className="text-xs text-zinc-500">Add Agent As Manager</label>
          <div className="flex flex-col gap-1 mt-1">
            <button
              onClick={() => handleCopy(space.managerJoinCode, true)}
              className="text-sm text-blue-400 hover:text-blue-300 text-left cursor-pointer"
            >
              [Copy Full Instructions]
            </button>
            <button
              onClick={() => navigator.clipboard.writeText(space.managerJoinCode)}
              className="text-sm text-blue-400 hover:text-blue-300 text-left cursor-pointer"
            >
              [Copy Code]
            </button>
          </div>
        </div>
        <div>
          <label className="text-xs text-zinc-500">Claw-Swarm Skill</label>
          <div className="mt-1">
            <button
              onClick={handleCopySkillInstructions}
              className="text-sm text-blue-400 hover:text-blue-300 cursor-pointer"
            >
              [Copy Skill Instructions]
            </button>
          </div>
        </div>
        <div>
          <button
            onClick={handleRegenerate}
            disabled={regenerating}
            className="text-xs text-zinc-400 hover:text-zinc-200 disabled:opacity-50 transition-colors"
          >
            {regenerating ? "Regenerating..." : "Regenerate codes"}
          </button>
        </div>
      </div>
    </div>
  );
}
