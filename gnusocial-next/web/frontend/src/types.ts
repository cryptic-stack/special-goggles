export type User = {
  id: string;
  username: string;
  displayName: string;
  bio: string;
  avatarUrl: string;
  domain: string | null;
  createdAt: string;
};

export type Post = {
  id: string;
  authorId: string;
  content: string;
  visibility: string;
  replyTo: string | null;
  federationId: string;
  createdAt: string;
};

