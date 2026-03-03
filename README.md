# gnusocial-modern

Modern GNU social-style platform with ActivityPub federation, local groups, and modern authentication.

## Current status

This repository is bootstrapped with:

- Docker Compose for Postgres, Redis, and API
- Go API skeleton with middleware and `/healthz`
- Config loader and storage connection adapters
- Startup migration runner (`backend/migrations/*.sql`)
- Dev seed for local actor `alice` (created when no local actors exist)
- ActivityPub discovery + actor endpoints (`webfinger`, `nodeinfo`, `users/:username`)
- Inbox ingestion with idempotency (`inbox_activities`) and basic activity handling
- Inbound HTTP Signature verification (with `AP_ALLOW_UNSIGNED_INBOUND` dev flag)
- Outbound HTTP Signature signing in delivery worker
- Actor collections + objects (`outbox`, `followers`, `following`, `notes/:id`)
- Local client API endpoints (`POST /api/v1/posts`, home/local timelines)
- Delivery worker with retry backoff over `deliveries` queue
- Local auth + sessions (`/auth/register`, `/auth/login`, `/auth/logout`, `/auth/me`)
- Follow/unfollow + reactions (`/api/v1/follows`, `/api/v1/unfollow`, likes/boosts)
- Local groups API (`create/join/post/timeline`)
- Notifications + basic moderation primitives (reports, domain policies)
- Modernized web UI wired to auth, posting, follows, groups, and notifications

## Quick start

1. Copy `.env.example` to `.env`.
2. Start services:

```bash
docker compose up --build
```

3. Verify API:

```bash
curl http://localhost:8080/healthz
```

4. Open the web UI:

```bash
http://localhost:8080/
```

5. Dev login (seeded):

```bash
username: alice
password: alice12345
```
You can change this with `DEV_SEED_PASSWORD` in `.env`.

## API surface (new)

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /auth/me`
- `POST /api/v1/follows`
- `POST /api/v1/unfollow`
- `POST /api/v1/notes/:id/like`
- `DELETE /api/v1/notes/:id/like`
- `POST /api/v1/notes/:id/boost`
- `DELETE /api/v1/notes/:id/boost`
- `DELETE /api/v1/posts/:id`
- `GET /api/v1/notifications`
- `POST /api/v1/notifications/read-all`
- `POST /api/v1/groups`
- `POST /api/v1/groups/:slug/join`
- `POST /api/v1/groups/:slug/posts`
- `GET /api/v1/groups/:slug/timeline`
- `POST /api/v1/reports`
- `GET /api/v1/admin/reports`
- `PUT /api/v1/admin/domain-policies`

## Production host policy

- In `APP_ENV=prod`, the API rejects localhost/loopback host access.
- In `APP_ENV=prod`, requests must target `APP_DOMAIN` (or forwarded host via `X-Forwarded-Host`).
