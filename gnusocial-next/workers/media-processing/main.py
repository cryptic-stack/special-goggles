import datetime as dt
import json
import os
import shutil
import time
import uuid
from pathlib import Path

import redis
from PIL import Image


STREAM_NAME = os.getenv("STREAM_NAME", "media_processing")
CONSUMER_GROUP = os.getenv("CONSUMER_GROUP", "media_workers")
CONSUMER_NAME = os.getenv("CONSUMER_NAME", "media-worker-1")
REDIS_HOST = os.getenv("REDIS_HOST", "redis")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))
MEDIA_ROOT = Path(os.getenv("MEDIA_ROOT", "/data/media"))


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


def virus_scan(path: Path) -> bool:
    # Placeholder for ClamAV integration.
    return path.exists()


def process_image(source: Path, target_dir: Path, target_name: str):
    destination = target_dir / target_name
    shutil.copy2(source, destination)

    try:
        with Image.open(destination) as img:
            resized = img.copy()
            resized.thumbnail((1920, 1920))
            resized.save(destination)

            thumb = img.copy()
            thumb.thumbnail((400, 400))
            thumb.save(target_dir / f"thumb-{target_name}")
    except Exception:  # noqa: BLE001
        # Keep non-image uploads untouched.
        pass

    return destination


def process_media(fields: dict):
    media_id = fields.get("media_id") or str(uuid.uuid4())
    source_path = fields.get("source_path")
    if not source_path:
        raise ValueError("source_path is required")

    source = Path(source_path)
    if not source.exists():
        raise FileNotFoundError(f"source file does not exist: {source}")

    if not virus_scan(source):
        raise RuntimeError("virus scan failed")

    now = dt.datetime.now(dt.UTC)
    target_dir = MEDIA_ROOT / str(now.year) / f"{now.month:02d}"
    target_dir.mkdir(parents=True, exist_ok=True)
    extension = source.suffix or ".bin"
    target_name = f"{media_id}{extension}"
    output = process_image(source, target_dir, target_name)
    log("info", "media_processed", media_id=media_id, output=str(output))


def main():
    client = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    ensure_consumer_group(client)
    log("info", "media_worker_started", stream=STREAM_NAME, consumer=CONSUMER_NAME)

    while True:
        entries = client.xreadgroup(CONSUMER_GROUP, CONSUMER_NAME, {STREAM_NAME: ">"}, count=20, block=5000)
        if not entries:
            continue

        for _, messages in entries:
            for message_id, fields in messages:
                try:
                    process_media(fields)
                except Exception as exc:  # noqa: BLE001
                    log("error", "media_processing_failed", message_id=message_id, error=str(exc))
                finally:
                    client.xack(STREAM_NAME, CONSUMER_GROUP, message_id)
        time.sleep(0.1)


if __name__ == "__main__":
    main()
