# Facebook Post Controls Gap Analysis

Last updated: March 4, 2026

## Scope
This compares `special-goggles` post/feed controls with Facebook-style post controls, focused on:
- Posting
- Hiding posts
- Mute/snooze behavior
- Follow/unfollow behavior

## Baseline references (Facebook)
- Audience selector supports options such as Public, Friends, Only Me, Custom, and per-post audience changes.
- Feed controls include Hide post, Snooze for 30 days, and Unfollow source from feed.
- Unfollow hides feed posts without necessarily removing other social connection state.

## Current behavior in this repo

### Posting
Implemented:
- Create post with content, media, and `visibility` (`public`, `unlisted`, `followers`, `direct`).
- Delete post.
- Quote posts.

Gap vs Facebook-style controls:
- No `Only Me` equivalent in UX semantics.
- No `Custom` audience targeting (allow/exclude specific people/lists).
- No per-user default audience setting for future posts.
- No post audience edit flow after publish.

### Hiding
Implemented:
- Mute/block actor-level filtering.

Gap:
- No single-post hide action (viewer-local hide without muting source).
- No hidden-post management view (undo/show hidden items).

### Mute / Snooze
Implemented:
- Actor mute (indefinite), actor block.

Gap:
- No timed mute/snooze (e.g., 30-day snooze).
- No mute scope control (feed-only vs notifications-only).
- No snooze expiry visibility in UI.

### Follow / Unfollow
Implemented:
- Follow/unfollow relations (ActivityPub-aligned), including remote behaviors.

Gap:
- No explicit soft-unfollow feed preference separate from hard unfollow relation removal.
- No follow preference levels (e.g., prioritize/less often) at feed layer.

## Priority gap summary

1. Viewer-local hide post
2. Snooze duration support on mutes
3. Feed-level soft follow/unfollow preferences
4. Richer posting audience controls (default + per-post custom)

## Constraints specific to this project
- Must preserve ActivityPub correctness: hard follow/unfollow remains protocol-level behavior.
- Feed preferences should be local-only overlays and not break federation semantics.
- Backward compatibility needed for existing endpoints/UI.

## Reference links
- https://www.facebook.com/help/200190190666715
- https://www.facebook.com/help/268028706671439/
- https://www.facebook.com/help/www/538433456491590
- https://www.facebook.com/help/190078864497547
- https://www.facebook.com/help/146865632052048/
- https://www.facebook.com/help/371675846332829/
