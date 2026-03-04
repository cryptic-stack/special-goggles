# Implementation Plan: Facebook-Style Post Controls

Last updated: March 4, 2026

## Goal
Add Facebook-style feed controls without breaking ActivityPub protocol behavior.

## Phase 1: Post audience enhancements

### Schema
- `user_post_settings`
  - `user_id` (PK/FK)
  - `default_audience` (`public|followers|private|custom`)
  - `audience_options_json` (custom targeting)
- `notes`
  - add `audience_json` for richer per-post targeting metadata

### API
- `GET /api/v1/settings/posting`
- `PUT /api/v1/settings/posting`
- extend `POST /api/v1/posts` to accept optional `audience` object
- `PATCH /api/v1/posts/{id}/audience` (local-author only)

### UI
- Composer audience selector:
  - Public
  - Followers
  - Only Me (mapped to private local visibility)
  - Custom (phase 1 starts with metadata storage + validation)

## Phase 2: Hide individual posts (viewer-local)

### Schema
- `hidden_posts`
  - `viewer_actor_id`
  - `note_id`
  - `reason` (optional)
  - `created_at`
  - `expires_at` (nullable)
  - PK `(viewer_actor_id, note_id)`

### API
- `POST /api/v1/notes/{id}/hide`
- `DELETE /api/v1/notes/{id}/hide`
- `GET /api/v1/hidden-posts`

### Timeline semantics
- Exclude hidden posts in home/local/bookmark/group timeline queries using `NOT EXISTS hidden_posts`.

### UI
- Per-post menu action: `Hide post`
- Hidden-posts panel: `Restore`

## Phase 3: Timed mute (snooze)

### Schema
- alter `mutes`:
  - add `expires_at TIMESTAMPTZ NULL`
  - add `scope TEXT NOT NULL DEFAULT 'feed'` (`feed|notifications|all`)

### API
- extend `POST /api/v1/mutes` payload:
  - `target`
  - `duration_days` (optional, e.g. 30)
  - `scope` (optional)
- `GET /api/v1/mutes` returns active mutes with expiry.

### Runtime behavior
- Treat mute as active when `expires_at IS NULL OR expires_at > now()`.
- Timeline filtering and notifications filtering honor `scope`.

### UI
- Add `Snooze 30 days` action.
- Show mute expiry in safety list.

## Phase 4: Soft follow/unfollow at feed layer

### Schema
- `feed_follow_preferences`
  - `viewer_actor_id`
  - `target_actor_id`
  - `is_following BOOLEAN NOT NULL DEFAULT TRUE`
  - `priority TEXT NOT NULL DEFAULT 'normal'` (`normal|high|low`)
  - `created_at`, `updated_at`
  - PK `(viewer_actor_id, target_actor_id)`

### API
- `POST /api/v1/feed/follow`
- `POST /api/v1/feed/unfollow`
- `GET /api/v1/feed/follows`

### Behavior
- Keep existing `POST /api/v1/unfollow` as hard protocol unfollow (no behavior change).
- Feed timelines use preference overlay to hide subjects set to `is_following=false`.

### UI
- Per-post/source actions:
  - `Unfollow (feed only)`
  - `Follow in feed`
- Safety/follow management panel for current preferences.

## Phase 5: Tests and rollout

### Tests
- Migration tests for new tables/columns.
- API tests for authz and idempotency.
- Timeline integration tests for hide/snooze/soft-unfollow filtering.
- UI smoke checks for action buttons and list rendering.

### Rollout
1. Ship backend tables and read paths first (feature flags).
2. Enable write endpoints.
3. Enable UI controls.
4. Keep hard unfollow endpoint unchanged for federation correctness.

## Proposed migration sequence
- `0007_post_audience_controls.sql`
- `0008_hidden_posts.sql`
- `0009_mute_expiry_scope.sql`
- `0010_feed_follow_preferences.sql`

## Immediate first sprint (recommended)

1. Implement `hidden_posts` + hide/unhide endpoints + timeline filtering.
2. Add mute `duration_days` support with expiry logic.
3. Add feed soft-unfollow preferences and endpoint pair.

This sequence gives the fastest user-visible parity improvements for Facebook-style feed controls.
