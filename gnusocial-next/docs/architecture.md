# Architecture

## Services
- `services/api`: public API + ActivityPub endpoints (Spring Boot)
- `services/federation`: reserved for dedicated federation API/service logic
- `services/timeline`: reserved for timeline read optimization service
- `services/media`: reserved for media service API
- `services/auth`: reserved for auth/OIDC service

## Workers
- `workers/federation-delivery`: consumes `federation_delivery` Redis stream
- `workers/timeline-fanout`: consumes `timeline_events` Redis stream
- `workers/media-processing`: consumes `media_processing` Redis stream

## Queue Design
Redis Streams with consumer groups:
- Replayable events
- Horizontal worker scaling
- Explicit pending/retry handling

Implemented streams:
- `timeline_events`
- `federation_delivery`
- `media_processing`

## Storage
PostgreSQL is source of truth.
Media files stored on mounted local volume at `/data/media/{year}/{month}/uuid.ext`.

## API Surface (v1)
### Discovery
- `GET /.well-known/webfinger`
- `GET /.well-known/nodeinfo`
- `GET /.well-known/host-meta`
- `GET /nodeinfo/2.1`

### Actor
- `GET /users/{username}`
- `GET /users/{username}/followers`
- `GET /users/{username}/following`
- `GET /users/{username}/outbox`
- `GET /users/{username}/inbox`
- `POST /users/{username}/inbox`

### App API
- `POST /api/v1/users`
- `GET /api/v1/users/{username}`
- `POST /api/v1/status`
- `GET /api/v1/timeline/home`
- `GET /api/v1/timeline/public`
- `POST /api/v1/users/follow`
- `DELETE /api/v1/users/follow`
- `POST /api/v1/users/mute`
- `DELETE /api/v1/users/mute`
- `POST /api/v1/users/hide-post`

### Federation ingress/egress
- `POST /inbox`
- `GET /outbox`

## Proxy Topology
- Nginx is the only host-exposed entrypoint (`:80`)
- API, frontend, admin, workers, PostgreSQL, Redis communicate on an internal Docker network
- Frontend routes on `/`, admin routes on `/admin/`, API and ActivityPub routes proxied to API service

## Scaling Strategy
- Scale `api` and workers horizontally
- Keep API stateless
- Use timeline fanout for read-heavy loads
- Start with single PostgreSQL primary, then move to:
  - read replicas
  - partitioning by month
  - logical sharding as needed
