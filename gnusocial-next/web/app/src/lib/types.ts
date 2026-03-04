export type DensityMode = "comfortable" | "default" | "compact";
export type UserRole = "member" | "moderator" | "admin";
export type TimelineKind = "home" | "following" | "local" | "federated";

export type Post = {
  id: string;
  threadId: string;
  author: {
    id: string;
    name: string;
    handle: string;
    avatarUrl: string;
  };
  createdAt: string;
  contentHtml: string;
  visibility: "public" | "unlisted" | "followers" | "direct";
  cw?: string;
  language: string;
  media: Array<{
    id: string;
    url: string;
    previewUrl: string;
    width: number;
    height: number;
    alt: string;
  }>;
  stats: {
    likes: number;
    boosts: number;
    replies: number;
  };
  myInteractions: {
    liked: boolean;
    bookmarked: boolean;
    boosted: boolean;
  };
  parentId?: string;
};

export type FeedPage = {
  items: Post[];
  nextCursor: string | null;
};

export type ThreadPayload = {
  root: Post;
  ancestors: Post[];
  descendants: Post[];
};

export type NotificationItem = {
  id: string;
  type: "mention" | "follow" | "boost" | "like" | "moderation";
  actor: string;
  text: string;
  unread: boolean;
  createdAt: string;
};
