ALTER TABLE notes
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE IF NOT EXISTS notifications (
  id              BIGSERIAL PRIMARY KEY,
  user_actor_id   BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  kind            TEXT NOT NULL, -- follow|like|announce|reply|mention
  actor_id        BIGINT REFERENCES actors(id) ON DELETE SET NULL,
  note_id         BIGINT REFERENCES notes(id) ON DELETE CASCADE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  read_at         TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_dedupe
ON notifications (user_actor_id, kind, actor_id, COALESCE(note_id, 0));

CREATE INDEX IF NOT EXISTS idx_notifications_user_created
ON notifications (user_actor_id, created_at DESC);

CREATE TABLE IF NOT EXISTS media_attachments (
  id              BIGSERIAL PRIMARY KEY,
  actor_id        BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  storage_key     TEXT NOT NULL UNIQUE,
  content_type    TEXT NOT NULL,
  byte_size       BIGINT NOT NULL,
  original_name   TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS note_attachments (
  note_id         BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  attachment_id   BIGINT NOT NULL REFERENCES media_attachments(id) ON DELETE CASCADE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (note_id, attachment_id)
);

CREATE TABLE IF NOT EXISTS domain_policies (
  domain          TEXT PRIMARY KEY,
  policy          TEXT NOT NULL, -- allow|limit|block
  reason          TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS reports (
  id              BIGSERIAL PRIMARY KEY,
  reporter_actor_id BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  target_actor_id BIGINT REFERENCES actors(id) ON DELETE SET NULL,
  target_note_id  BIGINT REFERENCES notes(id) ON DELETE SET NULL,
  reason          TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'open', -- open|resolved|dismissed
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_reports_status_created
ON reports (status, created_at DESC);
