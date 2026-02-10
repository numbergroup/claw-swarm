"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { AuthGuard } from "@/components/auth-guard";
import { Header } from "@/components/header";
import { CreateSpaceModal } from "@/components/create-space-modal";
import { JoinSpaceModal } from "@/components/join-space-modal";
import * as api from "@/lib/api";
import type { BotSpace } from "@/lib/types";

function Dashboard() {
  const [spaces, setSpaces] = useState<BotSpace[]>([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [joinOpen, setJoinOpen] = useState(false);

  useEffect(() => {
    api
      .listBotSpaces()
      .then((data) => setSpaces(data ?? []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function handleCreated(space: BotSpace) {
    setSpaces((prev) => [space, ...prev]);
  }

  function handleJoined(space: BotSpace) {
    setSpaces((prev) => {
      if (prev.some((s) => s.id === space.id)) return prev;
      return [space, ...prev];
    });
  }

  return (
    <>
      <Header />
      <main className="max-w-7xl mx-auto px-4 py-8">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-xl font-semibold">Bot Spaces</h1>
          <div className="flex gap-3">
            <button
              onClick={() => setJoinOpen(true)}
              className="rounded border border-zinc-700 hover:border-zinc-500 px-4 py-2 text-sm text-zinc-300 transition-colors"
            >
              Join Space
            </button>
            <button
              onClick={() => setCreateOpen(true)}
              className="rounded bg-blue-600 hover:bg-blue-500 px-4 py-2 text-sm font-medium text-white transition-colors"
            >
              Create Space
            </button>
          </div>
        </div>

        {loading ? (
          <div className="text-zinc-400">Loading spaces...</div>
        ) : spaces.length === 0 ? (
          <div className="text-center py-16">
            <p className="text-zinc-400 mb-4">No bot spaces yet</p>
            <p className="text-zinc-500 text-sm">Create a space or join one with an invite code.</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {spaces.map((space) => (
              <Link
                key={space.id}
                href={`/spaces/${space.id}`}
                className="block rounded-lg border border-zinc-800 bg-zinc-900 p-4 hover:border-zinc-600 transition-colors"
              >
                <h2 className="font-medium text-zinc-100 mb-1">{space.name}</h2>
                {space.description && (
                  <p className="text-sm text-zinc-400 line-clamp-2">{space.description}</p>
                )}
                <p className="text-xs text-zinc-500 mt-3">
                  Created {new Date(space.createdAt).toLocaleDateString()}
                </p>
              </Link>
            ))}
          </div>
        )}
      </main>

      <CreateSpaceModal open={createOpen} onClose={() => setCreateOpen(false)} onCreated={handleCreated} />
      <JoinSpaceModal open={joinOpen} onClose={() => setJoinOpen(false)} onJoined={handleJoined} />
    </>
  );
}

export default function Home() {
  return (
    <AuthGuard>
      <Dashboard />
    </AuthGuard>
  );
}
