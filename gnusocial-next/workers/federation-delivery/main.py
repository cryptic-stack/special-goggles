import base64
import datetime as dt
import hashlib
import json
import os
import time
from email.utils import format_datetime
from urllib.parse import urlparse

import redis
import requests
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import padding


STREAM_NAME = os.getenv("STREAM_NAME", "federation_delivery")
CONSUMER_GROUP = os.getenv("CONSUMER_GROUP", "federation_workers")
CONSUMER_NAME = os.getenv("CONSUMER_NAME", "federation-worker-1")
REDIS_HOST = os.getenv("REDIS_HOST", "redis")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))
MAX_RETRIES = int(os.getenv("MAX_RETRIES", "8"))
PRIVATE_KEY_PEM = os.getenv("FEDERATION_PRIVATE_KEY", "")
KEY_ID = os.getenv("FEDERATION_KEY_ID", "gnusocial-next#main-key")


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


def digest_header(body: bytes) -> str:
    digest = hashlib.sha256(body).digest()
    return "SHA-256=" + base64.b64encode(digest).decode("utf-8")


def sign_headers(private_key_pem: str, target_url: str, date_value: str, digest_value: str) -> str:
    parsed = urlparse(target_url)
    request_target = f"post {parsed.path or '/'}"
    signing_string = f"(request-target): {request_target}\ndate: {date_value}\ndigest: {digest_value}"

    private_key = serialization.load_pem_private_key(private_key_pem.encode("utf-8"), password=None)
    signature = private_key.sign(
        signing_string.encode("utf-8"),
        padding.PKCS1v15(),
        hashes.SHA256(),
    )
    signature_b64 = base64.b64encode(signature).decode("utf-8")
    return (
        f'keyId="{KEY_ID}",algorithm="rsa-sha256",headers="(request-target) date digest",'
        f'signature="{signature_b64}"'
    )


def deliver(target: str, activity_payload: dict):
    body = json.dumps(activity_payload).encode("utf-8")
    date_value = format_datetime(dt.datetime.now(dt.UTC))
    digest_value = digest_header(body)
    headers = {
        "Content-Type": "application/activity+json",
        "Accept": "application/activity+json",
        "Date": date_value,
        "Digest": digest_value,
    }
    if PRIVATE_KEY_PEM.strip():
        headers["Signature"] = sign_headers(PRIVATE_KEY_PEM, target, date_value, digest_value)

    response = requests.post(target, data=body, headers=headers, timeout=10)
    response.raise_for_status()


def queue_retry(client: redis.Redis, payload_str: str, next_attempt: int):
    backoff_seconds = min(2**next_attempt, 60)
    time.sleep(backoff_seconds)
    client.xadd(STREAM_NAME, {"payload": payload_str, "attempt": str(next_attempt)})


def handle_message(client: redis.Redis, message_id: str, fields: dict):
    payload_str = fields.get("payload")
    attempt = int(fields.get("attempt", "0"))
    if not payload_str:
        log("warn", "missing_payload", message_id=message_id)
        client.xack(STREAM_NAME, CONSUMER_GROUP, message_id)
        return

    payload = json.loads(payload_str)
    targets = payload.get("targets", [])
    if not isinstance(targets, list):
        targets = []

    try:
        for target in targets:
            deliver(target, payload)
        log("info", "delivery_success", message_id=message_id, targets=len(targets))
    except Exception as exc:  # noqa: BLE001
        if attempt < MAX_RETRIES:
            next_attempt = attempt + 1
            log(
                "warn",
                "delivery_retry",
                message_id=message_id,
                attempt=attempt,
                next_attempt=next_attempt,
                error=str(exc),
            )
            queue_retry(client, payload_str, next_attempt)
        else:
            log("error", "delivery_failed_max_retries", message_id=message_id, error=str(exc))
    finally:
        client.xack(STREAM_NAME, CONSUMER_GROUP, message_id)


def main():
    client = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
    ensure_consumer_group(client)
    log("info", "federation_worker_started", stream=STREAM_NAME, consumer=CONSUMER_NAME)

    while True:
        entries = client.xreadgroup(CONSUMER_GROUP, CONSUMER_NAME, {STREAM_NAME: ">"}, count=10, block=5000)
        if not entries:
            continue

        for _, messages in entries:
            for message_id, fields in messages:
                handle_message(client, message_id, fields)


if __name__ == "__main__":
    main()

