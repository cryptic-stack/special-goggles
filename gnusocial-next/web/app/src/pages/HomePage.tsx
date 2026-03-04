import { FeedPageBase } from "./FeedPageBase";

export function HomePage() {
  return (
    <FeedPageBase
      title="Home"
      kind="home"
      emptyMessage="Your home feed is empty. Follow accounts or explore tags."
      enableHotkeys
    />
  );
}
