# Route Contracts

Each route defines data dependencies, filters, actions, shortcuts, and empty-state guidance.

## Home
- Data: mixed timeline
- Filters: following-only, language, media-only, hide boosts, hide replies
- Actions: compose, reply, boost, like, bookmark, mute thread, block
- Shortcuts: `J/K` navigate, `Enter` open, `R` reply, `B` bookmark, `M` mute thread, `/` search
- Empty state: "Your home feed is empty. Follow accounts or explore tags."

## Following
- Data: strictly chronological feed from followed accounts
- Filters: hide boosts, hide replies, language, media-only
- Actions and shortcuts: same as Home

## Local
- Data: local instance feed
- Filters: safety filters, language, media-only, hide replies, hide boosts
- Empty state: teaches local account discovery and tags

## Federated
- Data: global federated feed
- Filters: safety filters, language, media-only, hide replies, hide boosts
- Empty state: teaches remote discovery and curated topics

## Explore
- Data sections: trending tags, suggested accounts, curated topics, directories
- Actions: follow, add tag to watchlist, save search

## Search
- Data: people, posts, tags, instances
- Filters: time range, language, from-following, media-only
- Actions: follow, open thread, save search

## Notifications
- Tabs: Mentions, Follows, Boosts, Likes, Moderation
- Filters: unread, from-following, mentions-only
- Actions: mark read, bulk mark read, mute thread, block
- Shortcuts: `Shift+R` mark read, `X` bulk mode

## Lists
- Data: list management and list feeds
- Actions: create list, add/remove accounts, list feed filtering

## Bookmarks
- Data: bookmarked posts with folders/tags
- Actions: tag, move folder, bulk manage
- Empty state: "Bookmark posts with B. Organize here."

## Profile
- Tabs: Posts, Replies, Media, About
- Actions: follow, mute, block, add to list
- Metadata: verified links and profile fields

## Thread
- Data: context above + replies below
- Controls: collapse/expand subthreads, reader mode, mute conversation
- Shortcuts: `C` collapse, `E` expand, `R` reply, `M` mute

## Settings
- Data: account, appearance, privacy, filters/mutes, sessions, export
- Actions: density mode change and keyboard shortcut reference

## Moderation/Admin (Role Gated)
- Data queues: reports, flagged posts, domain blocks, audit logs
- Actions: bulk workflows with explicit impact copy
