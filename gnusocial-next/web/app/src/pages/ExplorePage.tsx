import { TopBar } from "../components/TopBar";

const trending = ["#fediverse", "#queuefirst", "#gnusocial", "#activitypub"];
const suggested = ["@alice@local", "@devops@infra.zone", "@books@library.social"];
const topics = ["Open source communities", "Local meetups", "Distributed moderation"];

export function ExplorePage() {
  return (
    <>
      <TopBar title="Explore" />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <h2 className="text-sm font-semibold">Trending Tags</h2>
          <div className="mt-2 flex flex-wrap gap-2">
            {trending.map((tag) => (
              <button key={tag} className="rounded-full border border-ui-line px-3 py-1 text-xs">
                {tag}
              </button>
            ))}
          </div>
        </section>
        <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <h2 className="text-sm font-semibold">Suggested Accounts</h2>
          <ul className="mt-2 space-y-2 text-sm">
            {suggested.map((name) => (
              <li key={name} className="flex items-center justify-between">
                <span>{name}</span>
                <button className="rounded-md border border-ui-line px-2 py-1 text-xs">Follow</button>
              </li>
            ))}
          </ul>
        </section>
        <section className="rounded-xl border border-ui-line bg-ui-panel p-4">
          <h2 className="text-sm font-semibold">Curated Topics</h2>
          <ul className="mt-2 space-y-2 text-sm text-ui-muted">
            {topics.map((topic) => (
              <li key={topic}>{topic}</li>
            ))}
          </ul>
        </section>
      </div>
    </>
  );
}
