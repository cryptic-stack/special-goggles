import { FeedPageBase } from "./FeedPageBase";

export function FederatedPage() {
  return (
    <FeedPageBase
      title="Federated"
      kind="federated"
      emptyMessage="The federated timeline is quiet right now. Explore tags or adjust safety filters."
    />
  );
}
