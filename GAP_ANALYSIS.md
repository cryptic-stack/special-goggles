# Feature Gap Analysis

Last updated: March 3, 2026

## Baseline and intent
This project targets a modern GNU social-style experience with ActivityPub interoperability and practical parity for core microblog workflows. The comparison baseline used here is:
- GNU social style: notices, groups, federation-first utility UI
- Modern microblog expectations (X-like): fast timeline UX, notifications, richer social controls, polished navigation

## Current parity snapshot

| Area | Current status in this project | Gap vs GNU social / modern networks |
|---|---|---|
| Auth and sessions | Username/email login, registration, logout, session cookie, `/auth/me` | Missing passkeys and OIDC/social login providers |
| Posting | Create note, visibility, media uploads, delete own posts | Missing edit history, drafts, scheduled posts, polls |
| Timelines | Local + Home timelines, cursor pagination, endless scrolling | Missing thread-focused view and ranked/discovery feed |
| Social actions | Follow/unfollow, like, boost, notifications | Missing bookmarks, quote-posts, mute/block, pinning |
| Groups | Create, join, post, group timeline | Missing moderation roles, invites, search/filter, discovery |
| Moderation | Report creation, admin report listing, domain policy updates | Missing user-level moderation queue tooling and appeal flow |
| Federation core | WebFinger, NodeInfo, actor/object endpoints, inbox/outbox, signatures, delivery retries | Missing richer federation diagnostics UI and relay tooling |
| Personalization | Per-user theme presets + custom colors + font/density/corners | Missing reusable shared themes and per-device settings |
| UI layout and menu | New dark GNU social-style shell + functional menu actions (jump to sections, switch local/home, refresh) | Missing keyboard command palette and customizable nav order |

## Menu and layout status

Implemented in this pass:
- Dark-first default visual style (`gnusocial` preset)
- Top main menu with functional actions:
  - Jump to Timeline, Compose, Groups, Notifications, Theme, Federation panels
  - Switch timeline mode (Local/Home)
  - Refresh all data
- Left rail menu mirror with same functional actions
- Dynamic federation links tied to current username (Actor/Outbox)
- Mobile menu toggle with collapsible top menu

Remaining menu/layout gaps:
- Saved per-user menu layout preferences (collapsed panels, order)
- Global keyboard shortcuts/help overlay
- Dedicated thread page layout with nested replies and context

## Priority implementation backlog (next)

1. Threading and conversation depth
2. Mute/block/bookmark primitives and APIs
3. Search (users, posts, groups)
4. Group moderation roles + member management
5. Passkeys and at least one OIDC provider
6. Federation diagnostics panel (delivery failures, signature errors, retries)
