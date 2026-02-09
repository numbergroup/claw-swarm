"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import type { BotSpace } from "@/lib/types";
import * as api from "@/lib/api";

interface Props {
  space: BotSpace;
  isOwner: boolean;
  onUpdated: (space: BotSpace) => void;
}

export function SpaceHeader({ space, isOwner, onUpdated }: Props) {
  const router = useRouter();
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(space.name);
  const [description, setDescription] = useState(space.description || "");
  const [saving, setSaving] = useState(false);

  async function handleSave(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const updated = await api.updateBotSpace(space.id, {
        name,
        description: description || undefined,
      });
      onUpdated(updated);
      setEditing(false);
    } catch {}
    setSaving(false);
  }

  async function handleDelete() {
    if (!confirm("Delete this space? This cannot be undone.")) return;
    try {
      await api.deleteBotSpace(space.id);
      router.push("/");
    } catch {}
  }

  if (editing) {
    return (
      <form onSubmit={handleSave} className="space-y-2">
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-1.5 text-sm text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
        <input
          type="text"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Description"
          className="w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-1.5 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
        <div className="flex gap-2">
          <button
            type="submit"
            disabled={saving}
            className="text-xs text-blue-400 hover:text-blue-300 disabled:opacity-50"
          >
            {saving ? "Saving..." : "Save"}
          </button>
          <button
            type="button"
            onClick={() => setEditing(false)}
            className="text-xs text-zinc-400 hover:text-zinc-200"
          >
            Cancel
          </button>
        </div>
      </form>
    );
  }

  return (
    <div className="flex items-start justify-between">
      <div>
        <h1 className="text-lg font-semibold">{space.name}</h1>
        {space.description && (
          <p className="text-sm text-zinc-400 mt-0.5">{space.description}</p>
        )}
      </div>
      {isOwner && (
        <div className="flex gap-2 shrink-0">
          <button
            onClick={() => setEditing(true)}
            className="text-xs text-zinc-400 hover:text-zinc-200 px-2 py-1 rounded hover:bg-zinc-800 transition-colors"
          >
            Edit
          </button>
          <button
            onClick={handleDelete}
            className="text-xs text-red-500 hover:text-red-400 px-2 py-1 rounded hover:bg-zinc-800 transition-colors"
          >
            Delete
          </button>
        </div>
      )}
    </div>
  );
}
