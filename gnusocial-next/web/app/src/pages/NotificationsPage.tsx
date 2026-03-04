import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import { Tabs } from "../components/Tabs";
import { TopBar } from "../components/TopBar";
import { useUiStore } from "../features/ui/store";
import { getNotifications } from "../lib/api";

export function NotificationsPage() {
  const pushToast = useUiStore((s) => s.pushToast);
  const [tab, setTab] = useState("Mentions");
  const [unreadOnly, setUnreadOnly] = useState(false);
  const [bulkMode, setBulkMode] = useState(false);
  const [selected, setSelected] = useState<Record<string, boolean>>({});

  const notifications = useQuery({
    queryKey: ["notifications"],
    queryFn: getNotifications
  });

  useEffect(() => {
    function onKeys(event: KeyboardEvent) {
      const key = event.key.toLowerCase();
      if (event.shiftKey && key === "r") {
        pushToast("Marked all visible notifications as read.");
      }
      if (key === "x") {
        setBulkMode((v) => !v);
      }
    }
    window.addEventListener("keydown", onKeys);
    return () => window.removeEventListener("keydown", onKeys);
  }, [pushToast]);

  const filtered = useMemo(() => {
    const rows = notifications.data ?? [];
    const tabType = tab.toLowerCase().slice(0, -1);
    return rows.filter((row) => {
      if (tab !== "All" && row.type !== tabType) {
        return false;
      }
      if (unreadOnly && !row.unread) {
        return false;
      }
      return true;
    });
  }, [notifications.data, tab, unreadOnly]);

  function bulkMarkRead() {
    const count = Object.values(selected).filter(Boolean).length;
    pushToast(count ? `Marked ${count} notification(s) read.` : "No notifications selected.");
  }

  return (
    <>
      <TopBar
        title="Notifications"
        controls={
          <button type="button" className="rounded-md border border-ui-line px-2 py-1 text-xs" onClick={() => pushToast("Marked all as read.")}>
            Mark all read
          </button>
        }
      />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        <Tabs tabs={["All", "Mentions", "Follows", "Boosts", "Likes", "Moderation"]} active={tab} onChange={setTab} />
        <div className="flex items-center gap-3 rounded-xl border border-ui-line bg-ui-panel p-3 text-sm">
          <label className="flex items-center gap-2">
            <input type="checkbox" checked={unreadOnly} onChange={(event) => setUnreadOnly(event.target.checked)} />
            Unread only
          </label>
          <button type="button" className={`rounded-md px-2 py-1 text-xs ${bulkMode ? "bg-ui-accent text-ui-bg" : "border border-ui-line"}`} onClick={() => setBulkMode((v) => !v)}>
            Bulk mode (X)
          </button>
          <button type="button" className="rounded-md border border-ui-line px-2 py-1 text-xs" onClick={bulkMarkRead}>
            Bulk mark read (Shift+R)
          </button>
        </div>
        <ul className="space-y-2">
          {filtered.map((row) => (
            <li key={row.id} className="rounded-xl border border-ui-line bg-ui-panel p-3 text-sm">
              <div className="flex items-start gap-2">
                {bulkMode ? (
                  <input
                    type="checkbox"
                    checked={Boolean(selected[row.id])}
                    onChange={(event) => setSelected((prev) => ({ ...prev, [row.id]: event.target.checked }))}
                  />
                ) : null}
                <div className="flex-1">
                  <p>
                    <strong>{row.actor}</strong> {row.text}
                  </p>
                  <p className="text-xs text-ui-muted">{new Date(row.createdAt).toLocaleString()}</p>
                </div>
                {row.unread ? <span className="rounded-full bg-ui-accent px-2 py-1 text-[10px] text-ui-bg">Unread</span> : null}
              </div>
            </li>
          ))}
        </ul>
      </div>
    </>
  );
}
