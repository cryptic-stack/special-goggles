import { Bookmark, Heart, MessageCircle, Repeat2 } from "lucide-react";
import { Link } from "react-router-dom";
import { useUiStore } from "../features/ui/store";
import type { Post } from "../lib/types";
import { MediaGrid } from "./MediaGrid";

type PostCardProps = {
  post: Post;
  selected?: boolean;
  onReply?: (post: Post) => void;
  onBookmark?: (post: Post) => void;
  onMuteThread?: (post: Post) => void;
};

export function PostCard({ post, selected = false, onReply, onBookmark, onMuteThread }: PostCardProps) {
  const density = useUiStore((s) => s.density);

  const cardDensity =
    density === "comfortable" ? "p-5" : density === "compact" ? "p-3 text-sm" : "p-4";

  return (
    <article
      id={`post-${post.id}`}
      className={`group rounded-xl border bg-ui-panel transition ${
        selected ? "border-ui-accent" : "border-ui-line hover:border-ui-muted/40"
      } ${cardDensity}`}
    >
      <div className="flex items-start gap-3">
        <img
          src={post.author.avatarUrl}
          alt={`${post.author.name} avatar`}
          className={`rounded-full border border-ui-line ${density === "compact" ? "h-8 w-8" : "h-10 w-10"}`}
        />
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <div className="truncate">
              <span className="font-semibold">{post.author.name}</span>
              <span className="ml-2 text-sm text-ui-muted">{post.author.handle}</span>
            </div>
            <time className="shrink-0 text-xs text-ui-muted">{new Date(post.createdAt).toLocaleString()}</time>
          </div>
          {post.cw ? <p className="mt-2 text-xs font-medium uppercase tracking-wide text-ui-accent">CW: {post.cw}</p> : null}
          <p className="mt-2 whitespace-pre-wrap text-sm leading-6" dangerouslySetInnerHTML={{ __html: post.contentHtml }} />
          <MediaGrid media={post.media} density={density} />
          <div className="mt-3 flex items-center justify-between text-xs text-ui-muted opacity-0 transition group-hover:opacity-100">
            <span>Visibility: {post.visibility}</span>
            <span>Language: {post.language}</span>
          </div>
          <footer className="mt-3 flex items-center gap-2">
            <button
              aria-label="Reply to post"
              type="button"
              onClick={() => onReply?.(post)}
              className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-ui-muted hover:bg-ui-surface hover:text-ui-text"
            >
              <MessageCircle size={14} />
              {post.stats.replies}
            </button>
            <button aria-label="Boost post" type="button" className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-ui-muted hover:bg-ui-surface hover:text-ui-text">
              <Repeat2 size={14} />
              {post.stats.boosts}
            </button>
            <button aria-label="Like post" type="button" className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-ui-muted hover:bg-ui-surface hover:text-ui-text">
              <Heart size={14} />
              {post.stats.likes}
            </button>
            <button
              aria-label="Bookmark post"
              type="button"
              onClick={() => onBookmark?.(post)}
              className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-ui-muted hover:bg-ui-surface hover:text-ui-text"
            >
              <Bookmark size={14} />
              {post.myInteractions.bookmarked ? "Saved" : "Save"}
            </button>
            <button
              aria-label="Mute thread"
              type="button"
              onClick={() => onMuteThread?.(post)}
              className="ml-auto rounded-md px-2 py-1 text-xs text-ui-muted hover:bg-ui-surface hover:text-ui-text"
            >
              Mute
            </button>
            <Link to={`/thread/${post.threadId}`} className="rounded-md px-2 py-1 text-xs text-ui-muted hover:bg-ui-surface hover:text-ui-text">
              Open
            </Link>
          </footer>
        </div>
      </div>
    </article>
  );
}
