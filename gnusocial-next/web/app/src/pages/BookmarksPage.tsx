import { useMemo, useState } from "react";
import { EmptyState } from "../components/EmptyState";
import { TopBar } from "../components/TopBar";
import { useUiStore } from "../features/ui/store";

type BookmarkItem = { id: string; title: string; folder: string; tags: string[] };

const initialBookmarks: BookmarkItem[] = [
  { id: "b1", title: "Federation queue patterns", folder: "Research", tags: ["queue", "federation"] },
  { id: "b2", title: "UI density notes", folder: "Design", tags: ["ux", "density"] }
];

export function BookmarksPage() {
  const pushToast = useUiStore((s) => s.pushToast);
  const [bookmarks, setBookmarks] = useState(initialBookmarks);
  const [bulk, setBulk] = useState<Record<string, boolean>>({});
  const [activeFolder, setActiveFolder] = useState("All");
  const [tagInput, setTagInput] = useState("");

  const folders = useMemo(() => ["All", ...Array.from(new Set(bookmarks.map((b) => b.folder)))], [bookmarks]);
  const visible = useMemo(
    () => (activeFolder === "All" ? bookmarks : bookmarks.filter((b) => b.folder === activeFolder)),
    [activeFolder, bookmarks]
  );

  function moveSelected(folder: string) {
    const ids = Object.entries(bulk)
      .filter(([, checked]) => checked)
      .map(([id]) => id);
    if (!ids.length) {
      return;
    }
    setBookmarks((prev) => prev.map((bookmark) => (ids.includes(bookmark.id) ? { ...bookmark, folder } : bookmark)));
    pushToast(`Moved ${ids.length} bookmarks.`);
  }

  return (
    <>
      <TopBar title="Bookmarks" />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        <div className="flex flex-wrap gap-2">
          {folders.map((folder) => (
            <button
              key={folder}
              type="button"
              onClick={() => setActiveFolder(folder)}
              className={`rounded-md px-3 py-1 text-sm ${
                folder === activeFolder ? "bg-ui-accent text-ui-bg" : "border border-ui-line"
              }`}
            >
              {folder}
            </button>
          ))}
          <button type="button" onClick={() => moveSelected("Inbox")} className="ml-auto rounded-md border border-ui-line px-3 py-1 text-sm">
            Move selected to Inbox
          </button>
        </div>

        {visible.length === 0 ? (
          <EmptyState title="Bookmarks are empty" description="Bookmark posts with B. Organize here." />
        ) : (
          <ul className="space-y-2">
            {visible.map((item) => (
              <li key={item.id} className="rounded-xl border border-ui-line bg-ui-panel p-3">
                <div className="flex items-start gap-2">
                  <input
                    type="checkbox"
                    checked={Boolean(bulk[item.id])}
                    onChange={(event) => setBulk((prev) => ({ ...prev, [item.id]: event.target.checked }))}
                  />
                  <div className="flex-1">
                    <p className="font-medium">{item.title}</p>
                    <p className="text-xs text-ui-muted">Folder: {item.folder}</p>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {item.tags.map((tag) => (
                        <span key={tag} className="rounded-full border border-ui-line px-2 py-1 text-xs">
                          {tag}
                        </span>
                      ))}
                    </div>
                    <div className="mt-2 flex gap-2">
                      <input
                        value={tagInput}
                        onChange={(event) => setTagInput(event.target.value)}
                        placeholder="Add tag"
                        className="rounded-md border border-ui-line bg-ui-surface px-2 py-1 text-xs"
                      />
                      <button
                        type="button"
                        onClick={() => {
                          if (!tagInput.trim()) {
                            return;
                          }
                          setBookmarks((prev) =>
                            prev.map((bookmark) =>
                              bookmark.id === item.id ? { ...bookmark, tags: [...bookmark.tags, tagInput.trim()] } : bookmark
                            )
                          );
                          setTagInput("");
                        }}
                        className="rounded-md border border-ui-line px-2 py-1 text-xs"
                      >
                        Tag
                      </button>
                    </div>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </>
  );
}
