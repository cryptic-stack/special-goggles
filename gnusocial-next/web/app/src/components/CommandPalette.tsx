import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useUiStore } from "../features/ui/store";

const baseCommands = [
  { label: "Go to Home", to: "/home" },
  { label: "Go to Following", to: "/following" },
  { label: "Go to Explore", to: "/explore" },
  { label: "Go to Search", to: "/search" },
  { label: "Open Notifications", to: "/notifications" },
  { label: "Open Lists", to: "/lists" },
  { label: "Open Bookmarks", to: "/bookmarks" },
  { label: "Open Settings", to: "/settings" },
  { label: "Open UI Kit", to: "/ui-kit" }
];

export function CommandPalette() {
  const open = useUiStore((s) => s.commandOpen);
  const setOpen = useUiStore((s) => s.setCommandOpen);
  const [query, setQuery] = useState("");
  const navigate = useNavigate();

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [setOpen]);

  const commands = useMemo(() => {
    if (!query.trim()) {
      return baseCommands;
    }
    const q = query.toLowerCase();
    return baseCommands.filter((cmd) => cmd.label.toLowerCase().includes(q));
  }, [query]);

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/50 p-4">
      <div className="mx-auto mt-14 max-w-2xl rounded-xl border border-ui-line bg-ui-surface p-3 shadow-2xl">
        <input
          autoFocus
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search routes, actions, people, tags..."
          className="w-full rounded-lg border border-ui-line bg-ui-panel px-3 py-2 text-sm"
        />
        <ul className="mt-2 max-h-80 overflow-auto">
          {commands.map((cmd) => (
            <li key={cmd.to}>
              <button
                type="button"
                onClick={() => {
                  navigate(cmd.to);
                  setOpen(false);
                  setQuery("");
                }}
                className="w-full rounded-md px-3 py-2 text-left text-sm hover:bg-ui-panel"
              >
                {cmd.label}
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
