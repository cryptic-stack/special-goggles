CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- users
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username TEXT NOT NULL,
  domain TEXT NULL,
  email TEXT NULL,
  password_hash TEXT NULL,
  display_name TEXT NOT NULL DEFAULT '',
  bio TEXT NOT NULL DEFAULT '',
  avatar_url TEXT NOT NULL DEFAULT '',
  public_key TEXT NOT NULL DEFAULT '',
  private_key TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (username, domain),
  UNIQUE (email)
);

-- posts
CREATE TABLE IF NOT EXISTS posts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'public',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  reply_to UUID NULL REFERENCES posts(id) ON DELETE SET NULL,
  federation_id TEXT NULL UNIQUE
);

-- follows
CREATE TABLE IF NOT EXISTS follows (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  followed_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (follower_id, followed_id)
);

-- mutes
CREATE TABLE IF NOT EXISTS mutes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  muter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  muted_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (muter_id, muted_id)
);

-- hidden posts
CREATE TABLE IF NOT EXISTS post_hides (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, post_id)
);

-- activities
CREATE TABLE IF NOT EXISTS activities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  type TEXT NOT NULL,
  actor_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
  object_id UUID NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- media
CREATE TABLE IF NOT EXISTS media (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  post_id UUID NULL REFERENCES posts(id) ON DELETE CASCADE,
  path TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- precomputed timeline
CREATE TABLE IF NOT EXISTS timeline_entries (
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, post_id)
);

CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_author ON posts (author_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_follows_followed ON follows (followed_id);
CREATE INDEX IF NOT EXISTS idx_mutes_muter ON mutes (muter_id);
CREATE INDEX IF NOT EXISTS idx_post_hides_user ON post_hides (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_timeline_entries_user_created ON timeline_entries (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_activities_created ON activities (created_at DESC);
