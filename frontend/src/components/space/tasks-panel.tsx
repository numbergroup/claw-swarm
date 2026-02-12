import type { SpaceTask, Bot } from "@/lib/types";

const statusStyles: Record<string, string> = {
  available: "bg-zinc-700 text-zinc-300",
  in_progress: "bg-blue-600/30 text-blue-400",
  completed: "bg-green-600/30 text-green-400",
  blocked: "bg-red-600/30 text-red-400",
};

const statusLabels: Record<string, string> = {
  available: "Available",
  in_progress: "In Progress",
  completed: "Completed",
  blocked: "Blocked",
};

export function TasksPanel({
  tasks,
  bots,
}: {
  tasks: SpaceTask[];
  bots: Bot[];
}) {
  if (tasks.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-zinc-500">No tasks yet</p>
      </div>
    );
  }

  const botMap = new Map(bots.map((b) => [b.id, b.name]));

  return (
    <div className="p-4 overflow-y-auto h-full">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {tasks.map((task) => (
          <div
            key={task.id}
            className="rounded-lg bg-zinc-800/80 border border-zinc-700/50 p-4 flex flex-col gap-2"
          >
            <div className="flex items-start justify-between gap-2">
              <h3 className="font-semibold text-zinc-100">{task.name}</h3>
              <span
                className={`text-xs px-2 py-0.5 rounded-full whitespace-nowrap ${statusStyles[task.status] ?? "bg-zinc-700 text-zinc-300"}`}
              >
                {statusLabels[task.status] ?? task.status}
              </span>
            </div>
            <p className="text-sm text-zinc-300 line-clamp-3">
              {task.description}
            </p>
            {task.botId && (
              <p className="text-xs text-zinc-500">
                Assigned to {botMap.get(task.botId) ?? "unknown bot"}
              </p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
