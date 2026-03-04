import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { useParams } from "react-router-dom";
import { Composer } from "../components/Composer";
import { SkeletonPost } from "../components/Skeletons";
import { ThreadView } from "../components/ThreadView";
import { TopBar } from "../components/TopBar";
import { useUiStore } from "../features/ui/store";
import { getThread } from "../lib/api";

export function ThreadPage() {
  const params = useParams();
  const threadId = params.threadId ?? "t-1";
  const pushToast = useUiStore((s) => s.pushToast);

  const thread = useQuery({
    queryKey: ["thread", threadId],
    queryFn: () => getThread(threadId)
  });

  useEffect(() => {
    function onKeys(event: KeyboardEvent) {
      const key = event.key.toLowerCase();
      if (key === "c") {
        pushToast("Collapse subthreads.");
      }
      if (key === "e") {
        pushToast("Expand subthreads.");
      }
      if (key === "r") {
        pushToast("Thread reply composer focused.");
      }
      if (key === "m") {
        pushToast("Conversation muted.");
      }
    }
    window.addEventListener("keydown", onKeys);
    return () => window.removeEventListener("keydown", onKeys);
  }, [pushToast]);

  return (
    <>
      <TopBar title="Thread" />
      <div className="mx-auto max-w-feed space-y-4 p-4">
        <Composer threadMode />
        {thread.isLoading ? <SkeletonPost /> : null}
        {thread.data ? <ThreadView thread={thread.data} /> : null}
      </div>
    </>
  );
}
