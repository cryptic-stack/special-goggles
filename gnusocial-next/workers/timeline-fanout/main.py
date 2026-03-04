import datetime as dt
import json
import os
import time

import psycopg
import redis


STREAM_NAME = os.getenv("STREAM_NAME", "timeline_events")
CONSUMER_GROUP = os.getenv("CONSUMER_GROUP", "timeline_workers")
CONSUMER_NAME = os.getenv("CONSUMER_NAME", "timeline-worker-1")
REDIS_HOST = os.getenv("REDIS_HOST", "redis")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://gnusocial:password@postgres:5432/gnusocial")


def log(level: str, message: str, **kwargs):
    payload = {
        "ts": dt.datetime.now(dt.UTC).isoformat(),
        "level": level,
        "message": message,
        **kwargs,
    }
    print(json.dumps(payload), flush=True)


def ensure_consumer_group(client: redis.Redis):
    try:
        client.xgroup_create(STREAM_NAME, CONSUMER_GROUP, id="0-0", mkstream=True)
        log("info", "created_consumer_group", stream=STREAM_NAME, group=CONSUMER_GROUP)
    except redis.ResponseError as exc:
        if "BUSYGROUP" not in str(exc):
            raise


def fanout_post(conn: psycopg.Connection, post_id: str, author_id: str):
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO timeline_entries (user_id, post_id, created_at)
            VALUES (%s::uuid, %s::uuid, now())
            ON CONFLICT (user_id, post_id) DO NOTHING
            """,
            (author_id, post_id),
        )
        cur.execute(
            """
            INSERT INTO timeline_entries (user_id, post_id, created_at)
            SELECT follower_id, %s::uuid, now()
            FROM follows
            WHERE followed_id = %s::uuid
            ON CONFLICT (user_id, post_id) DO NOTHING
            """,
            (post_id, author_id),
        )
    conn.commit()


def main():
    client = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    ensure_consumer_group(client)
    log("info", "timeline_worker_started", stream=STREAM_NAME, consumer=CONSUMER_NAME)

    while True:
        try:
            with psycopg.connect(DATABASE_URL, autocommit=False) as conn:
                while True:
                    entries = client.xreadgroup(
                        CONSUMER_GROUP, CONSUMER_NAME, {STREAM_NAME: ">"}, count=100, block=5000
                    )
                    if not entries:
                        continue

                    for _, messages in entries:
                        for message_id, fields in messages:
                            post_id = fields.get("post_id")
                            author_id = fields.get("author_id")
                            if not post_id or not author_id:
                                client.xack(STREAM_NAME, CONSUMER_GROUP, message_id)
                                continue
                            fanout_post(conn, post_id, author_id)
                            client.xack(STREAM_NAME, CONSUMER_GROUP, message_id)
                            log("info", "timeline_fanout_complete", message_id=message_id, post_id=post_id)
        except Exception as exc:  # noqa: BLE001
            log("error", "timeline_worker_error", error=str(exc))
            time.sleep(2)


if __name__ == "__main__":
    main()

