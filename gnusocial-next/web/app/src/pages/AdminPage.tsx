import { EmptyState } from "../components/EmptyState";
import { TopBar } from "../components/TopBar";
import { useUiStore } from "../features/ui/store";

export function AdminPage() {
  const role = useUiStore((s) => s.role);

  return (
    <>
      <TopBar title="Admin" />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        {role !== "admin" ? (
          <EmptyState title="Access denied" description="Administrator role required." />
        ) : (
          <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
            <h2 className="text-sm font-semibold">Admin Console</h2>
            <ul className="mt-2 space-y-2 text-sm text-ui-muted">
              <li>Domain policies</li>
              <li>Queue depth and throughput</li>
              <li>Audit exports</li>
            </ul>
          </section>
        )}
      </div>
    </>
  );
}
