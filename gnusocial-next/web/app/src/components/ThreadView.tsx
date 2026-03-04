import { useMemo, useState } from "react";
import type { ThreadPayload } from "../lib/types";
import { PostCard } from "./PostCard";

type ThreadViewProps = {
  thread: ThreadPayload;
};

export function ThreadView({ thread }: ThreadViewProps) {
  const [readerMode, setReaderMode] = useState(false);
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});

  const descendants = useMemo(() => {
    if (!readerMode) {
      return thread.descendants;
    }
    return thread.descendants.filter((d) => !d.parentId || d.parentId === thread.root.id);
  }, [readerMode, thread.descendants, thread.root.id]);

  function toggleCollapsed(id: string) {
    setCollapsed((prev) => ({ ...prev, [id]: !prev[id] }));
  }

  return (
    <section className="space-y-3">
      <div className="rounded-xl border border-ui-line bg-ui-panel p-3">
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setReaderMode((v) => !v)}
            className="rounded-md border border-ui-line px-3 py-1 text-xs"
          >
            {readerMode ? "Disable Reader Mode" : "Reader Mode"}
          </button>
          <button
            type="button"
            onClick={() => setCollapsed({})}
            className="rounded-md border border-ui-line px-3 py-1 text-xs"
          >
            Expand All
          </button>
          <button
            type="button"
            onClick={() =>
              setCollapsed(Object.fromEntries(thread.descendants.map((item) => [item.id, true])))
            }
            className="rounded-md border border-ui-line px-3 py-1 text-xs"
          >
            Collapse All
          </button>
        </div>
      </div>

      {thread.ancestors.map((ancestor) => (
        <div key={ancestor.id} className="border-l-2 border-ui-line pl-4">
          <PostCard post={ancestor} />
        </div>
      ))}

      <div className="border-l-2 border-ui-accent pl-4">
        <PostCard post={thread.root} />
      </div>

      {descendants.map((descendant) => (
        <div key={descendant.id} className="border-l-2 border-ui-line pl-4">
          <div className="mb-1 flex justify-end">
            <button
              type="button"
              onClick={() => toggleCollapsed(descendant.id)}
              className="rounded-md border border-ui-line px-2 py-1 text-xs text-ui-muted"
            >
              {collapsed[descendant.id] ? "Expand" : "Collapse"}
            </button>
          </div>
          {collapsed[descendant.id] ? null : <PostCard post={descendant} />}
        </div>
      ))}
    </section>
  );
}
