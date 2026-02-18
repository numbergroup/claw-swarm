import type { Artifact } from "@/lib/types";

function isUrl(str: string): boolean {
  try {
    const url = new URL(str);
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
}

interface Props {
  artifacts: Artifact[];
  hasMore: boolean;
  onLoadMore: () => void;
  loadingMore: boolean;
}

export function ArtifactsPanel({ artifacts, hasMore, onLoadMore, loadingMore }: Props) {
  if (artifacts.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-zinc-500">No artifacts yet</p>
      </div>
    );
  }

  return (
    <div className="p-4 overflow-y-auto h-full">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {artifacts.map((artifact) => (
          <div
            key={artifact.id}
            className="rounded-lg bg-zinc-800/80 border border-zinc-700/50 p-4 flex flex-col gap-2"
          >
            <h3 className="font-semibold text-zinc-100">{artifact.name}</h3>
            <p className="text-sm text-zinc-300 line-clamp-3">
              {artifact.description}
            </p>
            {isUrl(artifact.data) ? (
              <a
                href={artifact.data}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs text-blue-400 hover:text-blue-300 truncate"
              >
                {artifact.data}
              </a>
            ) : (
              <pre className="text-xs text-zinc-400 bg-zinc-900/50 rounded p-2 overflow-x-auto whitespace-pre-wrap break-words max-h-32">
                {artifact.data}
              </pre>
            )}
          </div>
        ))}
      </div>
      {hasMore && (
        <div className="flex justify-center mt-4">
          <button
            onClick={onLoadMore}
            disabled={loadingMore}
            className="rounded border border-zinc-700 px-4 py-2 text-sm text-zinc-300 hover:border-zinc-500 transition-colors disabled:opacity-50"
          >
            {loadingMore ? "Loading..." : "Load more"}
          </button>
        </div>
      )}
    </div>
  );
}
