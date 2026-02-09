"use client";

import { useState, type FormEvent } from "react";
import * as api from "@/lib/api";
import { ApiError } from "@/lib/api";
import type { BotSpace } from "@/lib/types";

interface Props {
  open: boolean;
  onClose: () => void;
  onCreated: (space: BotSpace) => void;
}

export function CreateSpaceModal({ open, onClose, onCreated }: Props) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  if (!open) return null;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      const space = await api.createBotSpace({
        name,
        description: description || undefined,
      });
      setName("");
      setDescription("");
      onCreated(space);
      onClose();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to create space");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60" onClick={onClose} />
      <div className="relative bg-zinc-900 border border-zinc-700 rounded-lg p-6 w-full max-w-md mx-4">
        <h2 className="text-lg font-semibold mb-4">Create Bot Space</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="bg-red-900/50 border border-red-700 text-red-300 text-sm rounded px-3 py-2">
              {error}
            </div>
          )}
          <div>
            <label htmlFor="space-name" className="block text-sm font-medium text-zinc-300 mb-1">
              Name
            </label>
            <input
              id="space-name"
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="My Bot Space"
            />
          </div>
          <div>
            <label htmlFor="space-desc" className="block text-sm font-medium text-zinc-300 mb-1">
              Description <span className="text-zinc-500">(optional)</span>
            </label>
            <textarea
              id="space-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="w-full rounded bg-zinc-800 border border-zinc-700 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
              placeholder="What is this space for?"
            />
          </div>
          <div className="flex gap-3 justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="rounded bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed px-4 py-2 text-sm font-medium text-white transition-colors"
            >
              {submitting ? "Creating..." : "Create"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
