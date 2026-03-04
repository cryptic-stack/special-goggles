import { EmptyState } from "../components/EmptyState";
import { TopBar } from "../components/TopBar";
import { useUiStore } from "../features/ui/store";

export function ModerationPage() {
  const role = useUiStore((s) => s.role);
  const allowed = role === "moderator" || role === "admin";

  return (
    <>
      <TopBar title="Moderation" />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        {!allowed ? (
          <EmptyState title="Access denied" description="Moderator role required." />
        ) : (
          <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
            <h2 className="text-sm font-semibold">Queues</h2>
            <ul className="mt-2 space-y-2 text-sm">
              <li>Reports queue</li>
              <li>Flagged posts</li>
              <li>Domain blocks</li>
              <li>Audit log</li>
            </ul>
            <button className="mt-3 rounded-md border border-ui-line px-3 py-1 text-xs">Bulk Resolve</button>
          </section>
        )}
      </div>
    </>
  );
}
