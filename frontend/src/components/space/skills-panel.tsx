import type { BotSkill } from "@/lib/types";

export function SkillsPanel({ skills }: { skills: BotSkill[] }) {
  if (skills.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-zinc-500">No skills registered yet</p>
      </div>
    );
  }

  return (
    <div className="p-4 overflow-y-auto h-full">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {skills.map((skill) => (
          <div
            key={skill.id}
            className="rounded-lg bg-zinc-800/80 border border-zinc-700/50 p-4 flex flex-col gap-2"
          >
            <div>
              <h3 className="font-semibold text-zinc-100">{skill.name}</h3>
              <p className="text-xs text-zinc-500">by {skill.botName}</p>
            </div>
            <p className="text-sm text-zinc-300 line-clamp-3">
              {skill.description}
            </p>
            {skill.tags && skill.tags.length > 0 && (
              <div className="flex flex-wrap gap-1.5 mt-1">
                {skill.tags.map((tag) => (
                  <span
                    key={tag}
                    className="text-xs px-2 py-0.5 rounded-full bg-zinc-700 text-zinc-300"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
