# UI-Facing Backend Contracts

To avoid layout jank and inconsistent interaction state, the UI expects these contracts from API services.

## Feed Endpoints
- Cursor pagination with stable ordering
- Deterministic cursors per filter set
- No duplicate IDs across paged responses

## Post Payload Shape
- `id`
- `author` object with display name, handle, and avatar
- `created_at`
- `content_html` (sanitized)
- `media[]` metadata:
  - `url`, `preview_url`, `width`, `height`, `alt`
- `visibility`
- `cw` (optional)
- `language`
- `stats` (`likes`, `boosts`, `replies`)
- `my_interactions` (`liked`, `bookmarked`, `boosted`)

## Thread Endpoint
- `root`
- `ancestors[]`
- `descendants[]`
- Parent references on descendants for spine rendering (`parent_id`)

## UI Stability Requirements
- Precomputed media dimensions to reserve rendering slots
- Sanitized HTML for content display
- Stable interaction flags so optimistic UI can reconcile correctly
