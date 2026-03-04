import { FormEvent, useMemo, useState } from "react";
import { Tabs } from "../components/Tabs";
import { TopBar } from "../components/TopBar";
import { useUiStore } from "../features/ui/store";

const SEARCH_KEY = "gnusocial-next:saved-searches";

export function SearchPage() {
  const pushToast = useUiStore((s) => s.pushToast);
  const [query, setQuery] = useState("");
  const [tab, setTab] = useState("People");
  const [saved, setSaved] = useState<string[]>(() => {
    const stored = localStorage.getItem(SEARCH_KEY);
    return stored ? JSON.parse(stored) : [];
  });
  const [filters, setFilters] = useState({
    mediaOnly: false,
    fromFollowing: false,
    language: "",
    timeRange: "7d"
  });

  const results = useMemo(() => {
    if (!query.trim()) {
      return [];
    }
    return Array.from({ length: 5 }, (_, i) => `${tab} result ${i + 1} for "${query}"`);
  }, [query, tab]);

  function onSubmit(event: FormEvent) {
    event.preventDefault();
    pushToast("Search executed.");
  }

  function saveSearch() {
    if (!query.trim()) {
      return;
    }
    const next = [query, ...saved.filter((s) => s !== query)].slice(0, 20);
    setSaved(next);
    localStorage.setItem(SEARCH_KEY, JSON.stringify(next));
    pushToast("Search saved locally.");
  }

  return (
    <>
      <TopBar title="Search" />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        <form onSubmit={onSubmit} className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <div className="flex gap-2">
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search people, posts, tags, instances"
              className="flex-1 rounded-lg border border-ui-line bg-ui-surface px-3 py-2"
            />
            <button type="submit" className="rounded-md bg-ui-accent px-3 py-2 text-sm font-semibold text-ui-bg">
              Search
            </button>
            <button type="button" onClick={saveSearch} className="rounded-md border border-ui-line px-3 py-2 text-sm">
              Save
            </button>
          </div>
          <div className="mt-3 grid grid-cols-2 gap-2 text-sm md:grid-cols-4">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={filters.mediaOnly}
                onChange={(event) => setFilters((f) => ({ ...f, mediaOnly: event.target.checked }))}
              />
              Media only
            </label>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={filters.fromFollowing}
                onChange={(event) => setFilters((f) => ({ ...f, fromFollowing: event.target.checked }))}
              />
              From following
            </label>
            <label>
              <span className="text-xs text-ui-muted">Language</span>
              <input
                value={filters.language}
                onChange={(event) => setFilters((f) => ({ ...f, language: event.target.value }))}
                className="ml-2 w-20 rounded-md border border-ui-line bg-ui-surface px-2 py-1"
              />
            </label>
            <label>
              <span className="text-xs text-ui-muted">Time</span>
              <select
                value={filters.timeRange}
                onChange={(event) => setFilters((f) => ({ ...f, timeRange: event.target.value }))}
                className="ml-2 rounded-md border border-ui-line bg-ui-surface px-2 py-1"
              >
                <option value="24h">24h</option>
                <option value="7d">7d</option>
                <option value="30d">30d</option>
              </select>
            </label>
          </div>
        </form>
        <Tabs tabs={["People", "Posts", "Tags", "Instances"]} active={tab} onChange={setTab} />
        <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <h2 className="text-sm font-semibold">Results</h2>
          <ul className="mt-2 space-y-2 text-sm">
            {results.length ? results.map((row) => <li key={row}>{row}</li>) : <li className="text-ui-muted">Type to search.</li>}
          </ul>
        </section>
        <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <h2 className="text-sm font-semibold">Saved Searches</h2>
          <div className="mt-2 flex flex-wrap gap-2">
            {saved.length ? (
              saved.map((entry) => (
                <button key={entry} className="rounded-full border border-ui-line px-3 py-1 text-xs" onClick={() => setQuery(entry)}>
                  {entry}
                </button>
              ))
            ) : (
              <span className="text-sm text-ui-muted">No saved searches yet.</span>
            )}
          </div>
        </section>
      </div>
    </>
  );
}
