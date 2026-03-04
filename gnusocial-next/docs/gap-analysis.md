# Feature Gap Analysis

## Baseline Comparison
### GNU social (legacy)
- Strong ActivityPub federation compatibility
- Plugin-based monolith with synchronous code paths
- Limited horizontal scaling ergonomics

### X/Twitter-style platforms
- Strong ranking/recommendation stack
- Rich notifications and engagement analytics
- Advanced search and trend detection

### Facebook-style platforms
- Fine-grained feed controls (hide/mute/unfollow)
- Rich relationship graph and groups/events
- Strong moderation and safety tooling

## Current `gnusocial-next` Coverage
- Posting: implemented
- Follow/unfollow: implemented
- Mute/unmute: implemented
- Hide post: implemented
- Public + home timelines: implemented
- ActivityPub inbox/outbox discovery basics: implemented
- Queue-driven fanout + federation worker: implemented

## Major Gaps Remaining
1. Auth and identity
- Missing production-grade session lifecycle, OIDC, and social login providers

2. Federation completeness
- Missing full HTTP Signature verification, key rotation, signed GET fetches, and shared inbox optimization

3. Timeline quality
- Missing ranking, recommendation, and anti-spam scoring

4. Notifications
- Missing dedicated notifications table, read state, and delivery channels

5. Search
- Missing indexed full-text + account discovery service

6. Media pipeline hardening
- Virus scanner integration and metadata extraction are placeholders

7. Moderation depth
- Missing block/report workflow, admin queueing, and policy automation

## Implementation Plan
### Phase 1 (next)
1. Add auth service (`services/auth`) with Redis-backed sessions and refresh flow
2. Add notifications table + worker + `/api/v1/notifications`
3. Add block/report endpoints and moderation queue stream

### Phase 2
1. Add OpenSearch/PostgreSQL FTS search service
2. Add federation signature verification and key caching
3. Add delivery dead-letter queue + replay tooling

### Phase 3
1. Add ranking service and personalized home timeline
2. Add anti-abuse heuristics and rate-limiting policies
3. Add analytics + trend service
