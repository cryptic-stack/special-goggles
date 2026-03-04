import { FeedPageBase } from "./FeedPageBase";

export function LocalPage() {
  return (
    <FeedPageBase
      title="Local"
      kind="local"
      emptyMessage="No local posts match your filters. Try changing language or media filters."
    />
  );
}
