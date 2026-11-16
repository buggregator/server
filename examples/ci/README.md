# Buggregator in CI

Run Buggregator next to your tests as a service container, point the app's
mail and SDKs at it, then assert through the HTTP API what the app sent.

| What | How |
|------|-----|
| Health check | `GET /api/version` (no auth) |
| Clean state before a test | `DELETE /api/events` — no body deletes all. JSON body: `type`, `project`, `keep_pinned` combine; `uuids` deletes only those events and ignores `type`/`project`. A body that is not valid JSON is 400 and deletes nothing |
| Wait for a mail | `GET /api/smtp/messages/wait?subject=...&to=...&timeout=10s` — 200 with the message, 408 on timeout |
| List events | `GET /api/events?type=sentry` (also `smtp`, `monolog`, `ray`, `var-dump`, ...) |
| Auth | Only when `auth.enabled`: header `X-Auth-Token: <jwt>` |

Full API: [`api/openapi.yaml`](../../api/openapi.yaml).

Point the app at the service:

- SMTP: host `buggregator` (GitLab) or `localhost` (GitHub), port `1025`
- Sentry DSN: `http://key@<host>:8000/1`
- Ray, VarDumper, Monolog, Inspector, Profiler: see the [docs](https://docs.buggregator.dev).

## GitHub Actions

```yaml
jobs:
  tests:
    runs-on: ubuntu-latest
    services:
      buggregator:
        image: ghcr.io/buggregator/server:latest
        ports: ["8000:8000", "1025:1025"]
    steps:
      - uses: actions/checkout@v4
      - run: ./run-your-tests # MAIL_HOST=localhost MAIL_PORT=1025 ...
```

A working job: [`.github/workflows/ci-example.yml`](../../.github/workflows/ci-example.yml).

## GitLab CI

See [`gitlab-ci.yml`](gitlab-ci.yml): the service alias `buggregator` is the
host name inside the job.

## The example test

[`smoke_test.py`](smoke_test.py) (Python 3 stdlib) waits for the server,
clears events, sends a mail and a Sentry event the way an app would, and
asserts both through the API. Env: `BUGGREGATOR_URL`, `BUGGREGATOR_SMTP`,
`BUGGREGATOR_TOKEN`.
