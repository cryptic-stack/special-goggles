-- Actors (local users + cached remote actors)
CREATE TABLE actors (
  id              BIGSERIAL PRIMARY KEY,
  local           BOOLEAN NOT NULL DEFAULT TRUE,
  username        TEXT,
  domain          TEXT,
  display_name    TEXT NOT NULL DEFAULT '',
  summary         TEXT NOT NULL DEFAULT '',
  actor_url       TEXT UNIQUE,        -- canonical ActivityPub Actor id (URL)
  inbox_url       TEXT,
  outbox_url      TEXT,
  followers_url   TEXT,
  following_url   TEXT,
  public_key_pem  TEXT,
  private_key_pem TEXT,               -- local only (encrypt later)
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (username, domain)
);

-- Notes (local + cached remote objects)
CREATE TABLE notes (
  id              BIGSERIAL PRIMARY KEY,
  local           BOOLEAN NOT NULL DEFAULT TRUE,
  note_url        TEXT UNIQUE,        -- canonical object id (URL)
  actor_id        BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  in_reply_to_url TEXT,
  content_html    TEXT NOT NULL,
  content_text    TEXT NOT NULL DEFAULT '',
  visibility      TEXT NOT NULL DEFAULT 'public', -- public|unlisted|followers|direct
  sensitive       BOOLEAN NOT NULL DEFAULT FALSE,
  published_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notes_actor_published ON notes(actor_id, published_at DESC);

-- Follow relations (local perspective)
CREATE TABLE follows (
  id                   BIGSERIAL PRIMARY KEY,
  follower_id          BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  following_id         BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  state                TEXT NOT NULL DEFAULT 'accepted', -- pending|accepted|rejected
  follow_activity_url  TEXT UNIQUE, -- activity.id for Follow
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (follower_id, following_id)
);

-- Likes / Announces
CREATE TABLE reactions (
  id              BIGSERIAL PRIMARY KEY,
  actor_id        BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  note_id         BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  kind            TEXT NOT NULL, -- like|announce
  activity_url    TEXT UNIQUE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (actor_id, note_id, kind)
);

-- Home timeline fan-out table (MVP scalability)
CREATE TABLE timeline_items (
  user_actor_id   BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  note_id         BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_actor_id, note_id)
);
CREATE INDEX idx_timeline_user_created ON timeline_items(user_actor_id, created_at DESC);

-- Groups (local first)
CREATE TABLE groups (
  id              BIGSERIAL PRIMARY KEY,
  local           BOOLEAN NOT NULL DEFAULT TRUE,
  slug            TEXT NOT NULL,
  title           TEXT NOT NULL,
  summary         TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (slug)
);

CREATE TABLE group_memberships (
  group_id        BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  actor_id        BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  role            TEXT NOT NULL DEFAULT 'member', -- owner|moderator|member
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (group_id, actor_id)
);

CREATE TABLE group_posts (
  group_id        BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  note_id         BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (group_id, note_id)
);

-- Inbox activity log (idempotency + forensic record)
CREATE TABLE inbox_activities (
  id              BIGSERIAL PRIMARY KEY,
  activity_id     TEXT UNIQUE,
  actor_url       TEXT,
  type            TEXT,
  raw_json        JSONB NOT NULL,
  received_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Outbound delivery queue (retryable)
CREATE TABLE deliveries (
  id              BIGSERIAL PRIMARY KEY,
  target_inbox    TEXT NOT NULL,
  activity_id     TEXT NOT NULL,
  activity_json   JSONB NOT NULL,
  state           TEXT NOT NULL DEFAULT 'queued', -- queued|sent|failed
  attempts        INT NOT NULL DEFAULT 0,
  last_error      TEXT NOT NULL DEFAULT '',
  next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_deliveries_next_attempt ON deliveries(state, next_attempt_at);

-- Auth: local users and external identities
CREATE TABLE users (
  id              BIGSERIAL PRIMARY KEY,
  actor_id        BIGINT NOT NULL UNIQUE REFERENCES actors(id) ON DELETE CASCADE,
  email           TEXT UNIQUE,
  password_hash   TEXT, -- nullable if social/passkey-only
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE identities (
  id                BIGSERIAL PRIMARY KEY,
  user_id           BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider          TEXT NOT NULL,     -- google|meta|x|tiktok
  provider_subject  TEXT NOT NULL,     -- stable subject/ID from provider
  email             TEXT,
  display_name      TEXT,
  avatar_url        TEXT,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (provider, provider_subject)
);

-- Sessions (server-side) (optional; JWT-only is ok too)
CREATE TABLE sessions (
  id              TEXT PRIMARY KEY,
  user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at      TIMESTAMPTZ NOT NULL
);

