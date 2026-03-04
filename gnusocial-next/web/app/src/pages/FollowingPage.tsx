import { FeedPageBase } from "./FeedPageBase";

export function FollowingPage() {
  return (
    <FeedPageBase
      title="Following"
      kind="following"
      emptyMessage="Follow local or federated accounts to populate this strictly chronological view."
    />
  );
}
