import type { FeedPage, NotificationItem, Post, ThreadPayload, TimelineKind } from "./types";

const baseAuthors = [
  { id: "u1", name: "Ari North", handle: "@arinorth", avatarUrl: "https://i.pravatar.cc/80?img=15" },
  { id: "u2", name: "Mina Vale", handle: "@minavale", avatarUrl: "https://i.pravatar.cc/80?img=24" },
  { id: "u3", name: "Theo Park", handle: "@theopark", avatarUrl: "https://i.pravatar.cc/80?img=8" },
  { id: "u4", name: "Devon Pike", handle: "@devonpike", avatarUrl: "https://i.pravatar.cc/80?img=67" }
];

function makePost(idx: number, kind: TimelineKind): Post {
  const author = baseAuthors[idx % baseAuthors.length];
  const date = new Date(Date.now() - idx * 1000 * 60 * 13).toISOString();
  return {
    id: `${kind}-${idx}`,
    threadId: `t-${Math.floor(idx / 4)}`,
    author,
    createdAt: date,
    contentHtml: `Queue-first architecture update #${idx}. <b>Stable ordering</b> and cursor pagination are now in review.`,
    visibility: "public",
    cw: idx % 7 === 0 ? "architecture spoilers" : undefined,
    language: idx % 2 ? "en" : "en-US",
    media:
      idx % 3 === 0
        ? [
            {
              id: `m-${idx}`,
              url: "https://images.unsplash.com/photo-1469474968028-56623f02e42e",
              previewUrl: "https://images.unsplash.com/photo-1469474968028-56623f02e42e?w=800&q=80",
              width: 1200,
              height: 800,
              alt: "Mountain landscape"
            }
          ]
        : [],
    stats: {
      likes: 12 + idx,
      boosts: 4 + (idx % 9),
      replies: 2 + (idx % 6)
    },
    myInteractions: {
      liked: idx % 5 === 0,
      bookmarked: idx % 4 === 0,
      boosted: idx % 6 === 0
    },
    parentId: idx % 5 === 0 ? `${kind}-${idx - 1}` : undefined
  };
}

const timelines: Record<TimelineKind, Post[]> = {
  home: Array.from({ length: 120 }, (_, i) => makePost(i + 1, "home")),
  following: Array.from({ length: 90 }, (_, i) => makePost(i + 1, "following")),
  local: Array.from({ length: 60 }, (_, i) => makePost(i + 1, "local")),
  federated: Array.from({ length: 140 }, (_, i) => makePost(i + 1, "federated"))
};

const notifications: NotificationItem[] = Array.from({ length: 30 }, (_, i) => ({
  id: `n-${i}`,
  type: (["mention", "follow", "boost", "like", "moderation"] as const)[i % 5],
  actor: baseAuthors[i % baseAuthors.length].name,
  text: `Notification event #${i + 1}`,
  unread: i % 3 !== 0,
  createdAt: new Date(Date.now() - i * 1000 * 60 * 9).toISOString()
}));

export async function fetchTimeline(kind: TimelineKind, cursor: string | null, limit = 20): Promise<FeedPage> {
  await delay(130);
  const rows = timelines[kind];
  const offset = cursor ? Number(cursor) : 0;
  const items = rows.slice(offset, offset + limit);
  const next = offset + limit < rows.length ? String(offset + limit) : null;
  return { items, nextCursor: next };
}

export async function fetchThread(threadId: string): Promise<ThreadPayload> {
  await delay(100);
  const seed = Number(threadId.replace(/\D/g, "")) || 1;
  const root = makePost(seed, "home");
  root.id = `thread-root-${seed}`;
  root.threadId = threadId;

  const ancestors = [1, 2].map((i) => {
    const p = makePost(seed + i + 200, "following");
    p.threadId = threadId;
    return p;
  });
  const descendants = [1, 2, 3, 4].map((i) => {
    const p = makePost(seed + i + 400, "local");
    p.threadId = threadId;
    p.parentId = i % 2 ? root.id : `thread-reply-${seed}-${i - 1}`;
    p.id = `thread-reply-${seed}-${i}`;
    return p;
  });

  return { root, ancestors, descendants };
}

export async function fetchNotifications(): Promise<NotificationItem[]> {
  await delay(90);
  return notifications;
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
