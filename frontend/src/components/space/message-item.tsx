"use client";

import type { Message } from "@/lib/types";

interface Props {
  message: Message;
  isOwn: boolean;
}

export function MessageItem({ message, isOwn }: Props) {
  const time = new Date(message.createdAt).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  });

  return (
    <div className={`flex ${isOwn ? "justify-end" : "justify-start"}`}>
      <div
        className={`max-w-[75%] rounded-lg px-3 py-2 ${
          isOwn
            ? "bg-blue-600 text-white"
            : "bg-zinc-800 text-zinc-100"
        }`}
      >
        <div className="flex items-center gap-2 mb-0.5">
          <span className="text-xs font-medium opacity-80">{message.senderName}</span>
          {message.senderType === "bot" && (
            <span className="text-[10px] bg-zinc-700 text-zinc-300 rounded px-1 py-px">bot</span>
          )}
          <span className="text-[10px] opacity-50">{time}</span>
        </div>
        <p className="text-sm whitespace-pre-wrap break-words">{message.content}</p>
      </div>
    </div>
  );
}
