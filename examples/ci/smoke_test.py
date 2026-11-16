#!/usr/bin/env python3
"""Smoke test: the app under test sends a mail and a Sentry event to
Buggregator; the test asserts they arrived through Buggregator's HTTP API.

Replace the "app" part with your own test suite: point its SMTP and Sentry
DSN at Buggregator and keep the asserts. Python 3 stdlib only.

Env:
  BUGGREGATOR_URL   HTTP API, default http://localhost:8000
  BUGGREGATOR_SMTP  SMTP host:port, default localhost:1025
  BUGGREGATOR_TOKEN JWT, only when auth is enabled on the server
"""
import json
import os
import smtplib
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from email.message import EmailMessage

URL = os.environ.get("BUGGREGATOR_URL", "http://localhost:8000").rstrip("/")
SMTP_HOST, SMTP_PORT = os.environ.get("BUGGREGATOR_SMTP", "localhost:1025").rsplit(":", 1)
TOKEN = os.environ.get("BUGGREGATOR_TOKEN", "")


def api(method, path, body=None, headers=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(URL + path, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if TOKEN:
        req.add_header("X-Auth-Token", TOKEN)
    for k, v in (headers or {}).items():
        req.add_header(k, v)
    with urllib.request.urlopen(req, timeout=70) as resp:
        raw = resp.read()
        return json.loads(raw) if raw else None


def wait_ready(seconds=60):
    """Health check: GET /api/version answers once the server is up."""
    deadline = time.time() + seconds
    while True:
        try:
            print("buggregator", api("GET", "/api/version").get("version"))
            return
        except (urllib.error.URLError, ConnectionError, OSError):
            if time.time() > deadline:
                sys.exit(f"buggregator at {URL} is not up after {seconds}s")
            time.sleep(1)


def main():
    wait_ready()

    # Start from a clean state: no body deletes every event.
    api("DELETE", "/api/events")

    # --- the app under test ---------------------------------------------
    subject = f"Welcome {uuid.uuid4().hex[:8]}"
    msg = EmailMessage()
    msg["From"] = "app@example.com"
    msg["To"] = "user@example.com"
    msg["Subject"] = subject
    msg.set_content("Thanks for signing up.")
    with smtplib.SMTP(SMTP_HOST, int(SMTP_PORT), timeout=10) as smtp:
        smtp.send_message(msg)

    # A Sentry SDK would send this to the DSN http://key@<host>:8000/1.
    error = f"Payment failed {uuid.uuid4().hex[:8]}"
    api("POST", "/api/1/store/", {
        "event_id": uuid.uuid4().hex,
        "platform": "python",
        "level": "error",
        "message": error,
    }, {"X-Sentry-Auth": "Sentry sentry_version=7, sentry_key=key"})
    # ---------------------------------------------------------------------

    # Assert the mail: long-poll until it arrives (408 on timeout).
    q = urllib.parse.urlencode({"subject": subject, "to": "user@example.com", "timeout": "10s"})
    try:
        mail = api("GET", f"/api/smtp/messages/wait?{q}")
    except urllib.error.HTTPError as e:
        sys.exit(f"mail {subject!r} did not arrive: HTTP {e.code}")
    assert mail["payload"]["subject"] == subject, mail
    print("mail ok:", subject)

    # Assert the Sentry event.
    events = api("GET", "/api/events?type=sentry")["data"]
    assert len(events) == 1, f"want 1 sentry event, got {len(events)}"
    assert error in json.dumps(events[0]["payload"]), events[0]
    print("sentry ok:", error)


if __name__ == "__main__":
    main()
