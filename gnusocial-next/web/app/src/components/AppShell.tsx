import { SquarePen } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { useUiStore } from "../features/ui/store";
import { CommandPalette } from "./CommandPalette";
import { Composer } from "./Composer";
import { Drawer } from "./ModalDrawer";
import { SidebarNav } from "./SidebarNav";
import { Toasts } from "./Toasts";

export function AppShell() {
  const location = useLocation();
  const navigate = useNavigate();
  const setCommandOpen = useUiStore((s) => s.setCommandOpen);
  const [composerOpen, setComposerOpen] = useState(false);

  useEffect(() => {
    function onGlobalHotkeys(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      if (target?.tagName === "INPUT" || target?.tagName === "TEXTAREA") {
        return;
      }

      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setCommandOpen(true);
      }
      if (event.key === "/") {
        event.preventDefault();
        navigate("/search");
      }
    }

    window.addEventListener("keydown", onGlobalHotkeys);
    return () => window.removeEventListener("keydown", onGlobalHotkeys);
  }, [navigate, setCommandOpen]);

  const railContent = useMemo(() => {
    if (location.pathname.startsWith("/thread")) {
      return "Thread tools: collapse/expand shortcuts C and E.";
    }
    if (location.pathname.startsWith("/search")) {
      return "Search tips: use saved searches and narrow with filters.";
    }
    if (location.pathname.startsWith("/notifications")) {
      return "Bulk actions: Shift+R to mark read, X to toggle bulk mode.";
    }
    return "Right rail is contextual and optional. Core flows stay in the center column.";
  }, [location.pathname]);

  return (
    <div className="min-h-screen bg-ui-bg text-ui-text">
      <div className="mx-auto grid min-h-screen max-w-[1600px] grid-cols-1 md:grid-cols-[260px_minmax(0,1fr)] xl:grid-cols-[280px_minmax(680px,760px)_320px]">
        <SidebarNav />
        <main className="relative border-x border-ui-line">
          <Outlet />
        </main>
        <aside className="hidden border-l border-ui-line bg-ui-surface p-4 xl:block">
          <div className="rounded-xl border border-ui-line bg-ui-panel p-4">
            <h2 className="text-sm font-semibold">Context Rail</h2>
            <p className="mt-2 text-sm text-ui-muted">{railContent}</p>
          </div>
        </aside>
      </div>

      <button
        aria-label="Compose new post"
        type="button"
        onClick={() => setComposerOpen(true)}
        className="fixed bottom-5 right-5 inline-flex items-center gap-2 rounded-full bg-ui-accent px-4 py-3 text-sm font-semibold text-ui-bg shadow-lg"
      >
        <SquarePen size={16} />
        Compose
      </button>

      <Drawer open={composerOpen} title="Compose" onClose={() => setComposerOpen(false)}>
        <Composer />
      </Drawer>
      <CommandPalette />
      <Toasts />
    </div>
  );
}
