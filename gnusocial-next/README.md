# gnusocial-next

Modern GNU Social platform designed for horizontal scaling, queue-driven federation, and container-native deployment.

## Core Principles
- Horizontally scalable stateless services
- Queue-first architecture (Redis Streams)
- Container-native operation
- Modular services
- PostgreSQL-first storage
- Local/distributed volume media storage (no object-storage dependency)

## Quick Start
```bash
cd gnusocial-next
docker compose up --build -d
```

Base URL: `http://localhost`

Public entrypoints are exposed through Nginx on port `80` only.

## Implemented Milestone
- Spring Boot API service with:
  - `/api/v1/health`
  - user registration + lookup
  - post creation
  - timeline APIs (public + home)
  - follow/unfollow, mute/unmute, hide-post
  - ActivityPub discovery and actor endpoints
  - inbox ingestion (`Create` activity parsing)
- Redis Streams fanout and federation event publishing
- Worker services:
  - `timeline-fanout`
  - `federation-delivery`
  - `media-processing`
- React frontend (dark modernized layout, infinite scrolling, composer, moderation controls)
- React admin scaffold (`/admin`)

## Monorepo Layout
See `docs/architecture.md` for details.

## Development
```bash
docker compose up --build -d
docker compose logs -f api
docker compose down
```

Scale workers:
```bash
docker compose up -d --scale federation-worker=10 --scale timeline-worker=5
```
