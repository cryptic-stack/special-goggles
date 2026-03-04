ALTER TABLE notes
ADD COLUMN IF NOT EXISTS quote_note_id BIGINT REFERENCES notes(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_notes_quote_note_id
ON notes (quote_note_id);

CREATE TABLE IF NOT EXISTS bookmarks (
  actor_id      BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  note_id       BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (actor_id, note_id)
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_actor_created
ON bookmarks (actor_id, created_at DESC);

CREATE TABLE IF NOT EXISTS mutes (
  actor_id         BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  target_actor_id  BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (actor_id, target_actor_id),
  CHECK (actor_id <> target_actor_id)
);

CREATE INDEX IF NOT EXISTS idx_mutes_actor_created
ON mutes (actor_id, created_at DESC);

CREATE TABLE IF NOT EXISTS blocks (
  actor_id         BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  target_actor_id  BIGINT NOT NULL REFERENCES actors(id) ON DELETE CASCADE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (actor_id, target_actor_id),
  CHECK (actor_id <> target_actor_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_actor_created
ON blocks (actor_id, created_at DESC);
