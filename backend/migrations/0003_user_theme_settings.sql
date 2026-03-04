CREATE TABLE IF NOT EXISTS user_theme_settings (
  user_id         BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  preset          TEXT NOT NULL DEFAULT 'forest',
  variables_json  JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
