# Feature Gap Analysis: X (Twitter) vs special-goggles

Last updated: March 4, 2026

## Scope
This analysis compares currently implemented product features in this repo against commonly used X product capabilities, then defines a concrete implementation sequence for missing items.

## Current project capability snapshot
Implemented today:
- Auth/session: register/login/logout, `/auth/me`
- Posts: create/delete, visibility, media attachments
- Timelines: home + local with cursor pagination and endless scrolling
- Social graph: follow/unfollow
- Reactions: like + boost (announce)
- Notifications: list + mark read
- Groups: create/join/post/timeline
- Moderation primitives: reports + domain policies
- Federation core: WebFinger, NodeInfo, AP actor/inbox/outbox/object, signature handling
- UI: dark GNU social-inspired layout, functional menu controls, per-user theme settings

## X feature parity matrix

| X feature area | Status in this repo | Gap summary |
|---|---|---|
| Home feed modes (For You / Following) | Partial | Home timeline exists, but no ranked discovery feed mode equivalent to For You |
| Repost + Quote | Partial | Repost exists (boost); quote-post is missing |
| Replies and conversation threading | Partial | Basic reply data exists; no dedicated thread view, reply tree UX, or conversation controls |
| Bookmarks | Missing | No bookmark schema/API/UI |
| Lists (curated feeds) | Missing | No list entities, memberships, or list timelines |
| Communities | Partial | Groups exist; missing moderation roles workflow, discovery, invites, and policy controls |
| Polls in posts | Missing | No poll schema/API/voting flow |
| Edit post | Missing | No edit window/version history |
| Scheduled posts / drafts | Missing | No scheduler or draft storage |
| Direct Messages | Missing | No private thread/message system |
| Search (people/posts) | Missing | No indexed search endpoint/UI |
| Mute / Block | Missing | No user-level safety graph for mute/block |
| Pinned post/profile highlights | Missing | No profile pin metadata or endpoint |
| Long-form publishing | Missing | No long-form/article model beyond regular note body |
| Audio live (Spaces-equivalent) | Missing | No live audio primitives (out of near-term scope) |
| Fact-check/community-note style annotations | Missing | No annotation/moderation voting system |
| Multi-account power workflow | Missing | No account switching/workspace management in UI |

## Priority roadmap

Priority is based on user impact, implementation cost, and leverage for later features.

### P0 (ship first)
1. Quote-posts
2. Bookmarks
3. Mute / Block
4. Thread view API + UI

Why first:
- These close the largest day-to-day usage gap with X while reusing current post/timeline models.

### P1 (core product depth)
1. Lists + list timeline
2. Search (users/posts/groups)
3. Drafts + scheduled posts
4. Polls
5. Pin post on profile

Why second:
- These improve retention and organization, and unlock power-user workflows.

### P2 (advanced)
1. Communities parity upgrades (roles/invites/discovery)
2. Edit post with history and time window
3. Direct Messages
4. Long-form posts

Why third:
- Higher complexity and moderation/safety overhead; best after P0/P1 stabilize.

### P3 (defer unless explicitly requested)
1. Spaces-equivalent live audio
2. Community-note style annotation system
3. Monetization/subscription tooling

## Execution plan (implementation-level)

### Phase A: data model foundation
- Add migrations for:
  - `quote_posts` (or `notes.quote_note_id` foreign key)
  - `bookmarks` (`actor_id`, `note_id`, `created_at`)
  - `mutes`, `blocks` (`actor_id`, `target_actor_id`, `created_at`)
  - `lists`, `list_members`, `list_follows`
  - `draft_posts`, `scheduled_posts`
  - `polls`, `poll_options`, `poll_votes`
  - `profile_pins`
- Add indexes for timeline/search access patterns.

### Phase B: API surface additions
- Add endpoints:
  - `POST /api/v1/notes/{id}/quote`
  - `POST/DELETE /api/v1/notes/{id}/bookmark`
  - `GET /api/v1/bookmarks`
  - `POST/DELETE /api/v1/mutes`
  - `POST/DELETE /api/v1/blocks`
  - `GET /api/v1/threads/{id}`
  - `POST /api/v1/lists`, `POST /api/v1/lists/{id}/members`, `GET /api/v1/lists/{id}/timeline`
  - `GET /api/v1/search`
  - `POST /api/v1/drafts`, `GET /api/v1/drafts`, `POST /api/v1/scheduled`
  - `POST /api/v1/polls`, `POST /api/v1/polls/{id}/vote`
  - `POST /api/v1/profile/pin/{id}`, `DELETE /api/v1/profile/pin`
- Enforce authz/safety checks using current session principal model.

### Phase C: timeline and safety semantics
- Exclude blocked actors from home/local/list timelines.
- Exclude muted actors from notifications and optional timeline filters.
- Add notification kinds for quote/bookmark/poll events where applicable.

### Phase D: UI and menu parity
- Extend menu with:
  - Bookmarks
  - Lists
  - Search
  - Drafts/Scheduled
  - Safety (Mute/Block)
- Add thread view route and quote composer flow.
- Add timeline mode selector for Following vs Discovery when ranking exists.

### Phase E: tests and quality gates
- Add unit tests for validation and authorization for each new endpoint.
- Add integration tests for timeline filtering under block/mute rules.
- Add migration tests and rollback checks for all new schema changes.

## Recommended immediate start

Start with this first implementation slice (lowest risk, highest impact):
1. Quote-post schema + API + UI action button
2. Bookmark schema + API + bookmarks panel
3. Block/mute schema + API + timeline filtering middleware

This slice is the shortest path to noticeable X-parity for daily use.

## External references used for X capability baseline
- X Help Center: Bookmarks
  - https://help.x.com/en/using-x/bookmarks
- X Help Center: Mute and block
  - https://help.x.com/en/using-x/blocking-and-unblocking-accounts
- X Help Center: Communities
  - https://help.x.com/en/using-x/communities
- X Help Center: Lists
  - https://help.x.com/en/using-x/lists
- X Help Center: Reposts and quotes
  - https://help.x.com/en/using-x/repost
- X Help Center: Audio Spaces
  - https://help.x.com/en/using-x/spaces
- X Help Center: Article (long-form) writing
  - https://help.x.com/en/using-x/x-articles
- X Help Center: Community Notes
  - https://help.x.com/en/using-x/community-notes
- X Pro: Schedule posts
  - https://pro.x.com/en/using-x-pro/manage-your-account/how-to-schedule-posts-for-later-on-web-apps
