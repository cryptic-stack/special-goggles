import { TopBar } from "../components/TopBar";
import { useUiStore } from "../features/ui/store";
import type { DensityMode, UserRole } from "../lib/types";

const shortcuts = [
  ["Ctrl+K", "Command palette"],
  ["/", "Open search"],
  ["J/K", "Home feed navigation"],
  ["Enter", "Open selected thread"],
  ["B", "Bookmark selected post"],
  ["M", "Mute thread"],
  ["Shift+R", "Mark notifications read"],
  ["X", "Toggle notification bulk mode"],
  ["C/E", "Collapse or expand thread groups"]
];

export function SettingsPage() {
  const density = useUiStore((s) => s.density);
  const setDensity = useUiStore((s) => s.setDensity);
  const reducedMotion = useUiStore((s) => s.reducedMotion);
  const toggleReducedMotion = useUiStore((s) => s.toggleReducedMotion);
  const messagesEnabled = useUiStore((s) => s.messagesEnabled);
  const setMessagesEnabled = useUiStore((s) => s.setMessagesEnabled);
  const role = useUiStore((s) => s.role);
  const setRole = useUiStore((s) => s.setRole);

  return (
    <>
      <TopBar title="Settings" />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <h2 className="text-sm font-semibold">Appearance</h2>
          <div className="mt-2 flex gap-2">
            {(["comfortable", "default", "compact"] as DensityMode[]).map((mode) => (
              <button
                key={mode}
                type="button"
                onClick={() => setDensity(mode)}
                className={`rounded-md px-3 py-1 text-sm ${
                  density === mode ? "bg-ui-accent text-ui-bg" : "border border-ui-line"
                }`}
              >
                {mode}
              </button>
            ))}
          </div>
          <label className="mt-3 flex items-center gap-2 text-sm">
            <input type="checkbox" checked={reducedMotion} onChange={toggleReducedMotion} />
            Reduced motion
          </label>
        </section>

        <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <h2 className="text-sm font-semibold">Features</h2>
          <label className="mt-2 flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={messagesEnabled}
              onChange={(event) => setMessagesEnabled(event.target.checked)}
            />
            Enable messages
          </label>
          <div className="mt-3">
            <span className="text-xs text-ui-muted">Role gate simulation</span>
            <div className="mt-2 flex gap-2">
              {(["member", "moderator", "admin"] as UserRole[]).map((candidate) => (
                <button
                  key={candidate}
                  type="button"
                  onClick={() => setRole(candidate)}
                  className={`rounded-md px-3 py-1 text-xs ${
                    role === candidate ? "bg-ui-accent text-ui-bg" : "border border-ui-line"
                  }`}
                >
                  {candidate}
                </button>
              ))}
            </div>
          </div>
        </section>

        <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <h2 className="text-sm font-semibold">Keyboard Shortcuts</h2>
          <ul className="mt-2 space-y-2 text-sm">
            {shortcuts.map(([keys, label]) => (
              <li key={keys} className="flex items-center justify-between rounded-md border border-ui-line px-3 py-2">
                <kbd className="rounded border border-ui-line px-2 py-1 text-xs">{keys}</kbd>
                <span className="text-ui-muted">{label}</span>
              </li>
            ))}
          </ul>
        </section>

        <section className="rounded-xl border border-ui-line bg-ui-panel p-4 text-sm text-ui-muted">
          Privacy, sessions, mutes, export, and account management endpoints connect in the next API integration pass.
        </section>
      </div>
    </>
  );
}
