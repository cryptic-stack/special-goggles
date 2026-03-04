# X Parity Implementation Plan

Last updated: March 4, 2026

## Goal
Close the highest-impact product gaps against X while preserving ActivityPub-first architecture and moderation safety.

## Milestone 1: Quote + Bookmark + Safety Graph

### Schema
- `notes.quote_note_id BIGINT NULL REFERENCES notes(id)`
- `bookmarks(actor_id BIGINT, note_id BIGINT, created_at TIMESTAMPTZ, PRIMARY KEY(actor_id, note_id))`
- `mutes(actor_id BIGINT, target_actor_id BIGINT, created_at TIMESTAMPTZ, PRIMARY KEY(actor_id, target_actor_id))`
- `blocks(actor_id BIGINT, target_actor_id BIGINT, created_at TIMESTAMPTZ, PRIMARY KEY(actor_id, target_actor_id))`

### API
- `POST /api/v1/notes/{id}/quote`
- `POST /api/v1/notes/{id}/bookmark`
- `DELETE /api/v1/notes/{id}/bookmark`
- `GET /api/v1/bookmarks`
- `POST /api/v1/mutes`
- `DELETE /api/v1/mutes`
- `POST /api/v1/blocks`
- `DELETE /api/v1/blocks`

### Timeline semantics
- Hidden when blocked either direction
- Optional hide muted in home/local

### UI
- Add action buttons to each timeline item: `Quote`, `Bookmark`, `Mute`, `Block`
- Add bookmarks panel and safety panel in left rail + top menu

### Tests
- Authorization tests for all write endpoints
- Filtering tests for block/mute behavior
- Bookmark listing pagination tests

## Milestone 2: Lists + Search + Thread View

### Schema
- `lists(id, owner_actor_id, name, description, created_at)`
- `list_members(list_id, actor_id, created_at, PRIMARY KEY(list_id, actor_id))`
- Indexes for list timeline fanout/filtering

### API
- `POST /api/v1/lists`
- `POST /api/v1/lists/{id}/members`
- `DELETE /api/v1/lists/{id}/members/{actor_id}`
- `GET /api/v1/lists/{id}/timeline`
- `GET /api/v1/search?q=&type=people|posts|groups`
- `GET /api/v1/threads/{id}`

### UI
- New list management panel
- Global search field in masthead
- Thread detail panel with parent + replies

### Tests
- Search query validation and pagination
- List membership authz and dedupe
- Thread retrieval ordering and depth limits

## Milestone 3: Drafts + Scheduling + Polls + Pinning

### Schema
- `draft_posts`
- `scheduled_posts`
- `polls`, `poll_options`, `poll_votes`
- `profile_pins`

### API
- `POST/GET/DELETE /api/v1/drafts`
- `POST/GET/DELETE /api/v1/scheduled`
- `POST /api/v1/polls`, `POST /api/v1/polls/{id}/vote`
- `POST /api/v1/profile/pin/{id}`, `DELETE /api/v1/profile/pin`

### Jobs
- Scheduler worker to publish due posts with retries

### UI
- Draft and scheduled queue in compose area
- Poll composer module
- Pinned post slot in profile card

### Tests
- Scheduler idempotency tests
- Poll one-vote-per-actor enforcement
- Pin replacement semantics

## Milestone 4: Communities parity and DM baseline

### Communities
- Role workflow: owner/mod/member actions
- Invite and approval controls
- Group discovery/search

### Direct Messages baseline
- `dm_threads`, `dm_participants`, `dm_messages`
- Endpoints for thread list, message send, read markers
- Abuse controls and block integration

## Non-goals for now
- Spaces-equivalent live audio
- Community Notes ranking system
- Monetization/subscription tooling

## Release strategy
1. Ship milestone 1 behind feature flags per endpoint.
2. Enable for local users only first.
3. Roll out UI controls after backend stability tests pass.
4. Expand to federated interactions where protocol-compatible.
