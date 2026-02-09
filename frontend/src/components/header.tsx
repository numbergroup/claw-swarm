"use client";

import Link from "next/link";
import { useAuth } from "@/lib/auth-context";

export function Header() {
  const { user, logout } = useAuth();

  return (
    <header className="border-b border-zinc-800 bg-zinc-900">
      <div className="max-w-7xl mx-auto px-4 h-14 flex items-center justify-between">
        <Link href="/" className="text-lg font-semibold text-blue-400 hover:text-blue-300">
          Claw Swarm
        </Link>
        {user && (
          <div className="flex items-center gap-4">
            <span className="text-sm text-zinc-400">
              {user.displayName || user.email}
            </span>
            <button
              onClick={logout}
              className="text-sm text-zinc-400 hover:text-zinc-200 transition-colors"
            >
              Logout
            </button>
          </div>
        )}
      </div>
    </header>
  );
}
