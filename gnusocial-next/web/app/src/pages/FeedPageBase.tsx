import { useInfiniteQuery } from "@tanstack/react-query";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { FiltersBar } from "../components/FiltersBar";
import { EmptyState } from "../components/EmptyState";
import { PostCard } from "../components/PostCard";
import { SkeletonList } from "../components/Skeletons";
import { TopBar } from "../components/TopBar";
import { useUiStore } from "../features/ui/store";
import { getTimeline } from "../lib/api";
import type { TimelineKind } from "../lib/types";

type FeedPageBaseProps = {
  title: string;
  kind: TimelineKind;
  emptyMessage: string;
  enableHotkeys?: boolean;
};

export function FeedPageBase({ title, kind, emptyMessage, enableHotkeys = false }: FeedPageBaseProps) {
  const navigate = useNavigate();
  const pushToast = useUiStore((s) => s.pushToast);
  const [language, setLanguage] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [chips, setChips] = useState([
    { id: "following-only", label: "Following only", active: false },
    { id: "media-only", label: "Media only", active: false },
    { id: "hide-boosts", label: "Hide boosts", active: false },
    { id: "hide-replies", label: "Hide replies", active: false }
  ]);

  const timeline = useInfiniteQuery({
    queryKey: ["timeline", kind],
    initialPageParam: null as string | null,
    queryFn: ({ pageParam }) => getTimeline(kind, pageParam),
    getNextPageParam: (lastPage) => lastPage.nextCursor
  });

  const posts = useMemo(() => timeline.data?.pages.flatMap((page) => page.items) ?? [], [timeline.data?.pages]);
  const visiblePosts = useMemo(() => {
    return posts.filter((post) => {
      if (language && post.language !== language) {
        return false;
      }
      if (chips.find((c) => c.id === "media-only")?.active && post.media.length === 0) {
        return false;
      }
      return true;
    });
  }, [posts, chips, language]);

  useEffect(() => {
    if (!enableHotkeys) {
      return;
    }
    function onKeys(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      if (target?.tagName === "INPUT" || target?.tagName === "TEXTAREA") {
        return;
      }

      if (event.key.toLowerCase() === "j") {
        event.preventDefault();
        setSelectedIndex((index) => Math.min(index + 1, Math.max(visiblePosts.length - 1, 0)));
      }
      if (event.key.toLowerCase() === "k") {
        event.preventDefault();
        setSelectedIndex((index) => Math.max(index - 1, 0));
      }
      if (event.key === "Enter" && visiblePosts[selectedIndex]) {
        navigate(`/thread/${visiblePosts[selectedIndex].threadId}`);
      }
      if (event.key.toLowerCase() === "b" && visiblePosts[selectedIndex]) {
        pushToast("Bookmarked post.");
      }
      if (event.key.toLowerCase() === "r" && visiblePosts[selectedIndex]) {
        pushToast("Reply composer opened.");
      }
      if (event.key.toLowerCase() === "m" && visiblePosts[selectedIndex]) {
        pushToast("Thread muted.");
      }
    }

    window.addEventListener("keydown", onKeys);
    return () => window.removeEventListener("keydown", onKeys);
  }, [enableHotkeys, navigate, pushToast, selectedIndex, visiblePosts]);

  useEffect(() => {
    const target = document.getElementById(`post-${visiblePosts[selectedIndex]?.id}`);
    target?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, [selectedIndex, visiblePosts]);

  function onToggleChip(id: string) {
    setChips((current) => current.map((chip) => (chip.id === id ? { ...chip, active: !chip.active } : chip)));
  }

  function onSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    navigate("/search");
  }

  return (
    <>
      <TopBar
        title={title}
        controls={
          <form onSubmit={onSearchSubmit}>
            <input
              className="w-40 rounded-md border border-ui-line bg-ui-panel px-2 py-1 text-xs"
              placeholder="Sort/filter..."
              aria-label="View controls"
            />
          </form>
        }
      />
      <FiltersBar chips={chips} onToggleChip={onToggleChip} language={language} onLanguageChange={setLanguage} />
      <div className="mx-auto max-w-feed space-y-3 p-4">
        {timeline.isLoading ? <SkeletonList count={3} /> : null}
        {!timeline.isLoading && visiblePosts.length === 0 ? (
          <EmptyState title={`${title} is empty`} description={emptyMessage} />
        ) : null}
        {visiblePosts.map((post, index) => (
          <PostCard
            key={post.id}
            post={post}
            selected={enableHotkeys && selectedIndex === index}
            onReply={() => pushToast("Reply composer opened.")}
            onBookmark={() => pushToast("Saved to bookmarks.")}
            onMuteThread={() => pushToast("Thread muted.")}
          />
        ))}
        {timeline.hasNextPage ? (
          <button
            type="button"
            onClick={() => timeline.fetchNextPage()}
            className="w-full rounded-lg border border-ui-line bg-ui-panel px-3 py-2 text-sm"
          >
            Load more
          </button>
        ) : null}
      </div>
    </>
  );
}
