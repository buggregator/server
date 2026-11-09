# Buggregator — one local server for everything your app sends while you develop

[![Release](https://img.shields.io/github/v/release/buggregator/server)](https://github.com/buggregator/server/releases/latest)
[![CI](https://github.com/buggregator/server/actions/workflows/ci.yml/badge.svg)](https://github.com/buggregator/server/actions/workflows/ci.yml)
[![Docker pulls](https://img.shields.io/docker/pulls/butschster/buggregator.svg)](https://hub.docker.com/r/butschster/buggregator)
<a href="https://discord.gg/vDsCD3EKUB"><img src="https://img.shields.io/badge/discord-chat-magenta.svg" alt="Discord"></a>
[![Follow on X](https://img.shields.io/badge/follow-%40buggregator-black?logo=x)](https://x.com/buggregator)

**One ~40 MB binary catches the errors, dumps, emails, logs, profiles and HTTP requests your app sends in development — through the SDKs and protocols you already use. No account, no cloud, no Buggregator package in your app.**

- **Errors and traces** from your Sentry SDK — exceptions, transactions, logs, metrics
- **Dumps** from Symfony VarDumper and Spatie Ray
- **Emails** from a fake SMTP server, with an API to wait for a message and pull links or codes in E2E tests
- **Logs** from Monolog
- **Profiles** from XHProf — call graph, flame chart, top functions
- **HTTP**: catch any request, proxy outgoing calls, or answer with a **mock server** (rules or an OpenAPI spec)
- **Xdebug in the browser**: pause, step, read variables — with or without an IDE
- **AI agents** read all of it over MCP

Mail, HTTP dumps, the proxy, mocks and MCP work with any language. The Sentry part speaks the Sentry protocol and is tested with the Sentry PHP SDK.

## Run it in 30 seconds

**Binary** (Linux, macOS, Windows; x86-64 and ARM):

```bash
curl -fsSL https://raw.githubusercontent.com/buggregator/server/master/scripts/install.sh | sh
buggregator
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/buggregator/server/master/scripts/install.ps1 | iex
```

The script picks the file for your system, checks its checksum and installs it without `sudo`. By default the binary listens on all network interfaces; to keep it on your machine only, see [Listen only on this machine](https://docs.buggregator.dev/install.html).

**Docker:**

```bash
docker run --pull always \
  -p 127.0.0.1:8000:8000 \
  -p 127.0.0.1:1025:1025 \
  -p 127.0.0.1:9912:9912 \
  -p 127.0.0.1:9913:9913 \
  ghcr.io/buggregator/server:latest
```

Open http://127.0.0.1:8000. The UI stays empty until your app sends something.

Events live in memory and are gone after a restart. To keep them: `buggregator -db ./buggregator.db`.

## Point your app at it

Change addresses, not code. Laravel / Symfony `.env`:

```env
SENTRY_LARAVEL_DSN=http://sentry@127.0.0.1:8000/1
MAIL_HOST=127.0.0.1
MAIL_PORT=1025
VAR_DUMPER_SERVER=127.0.0.1:9912
LOG_SOCKET_URL=127.0.0.1:9913
RAY_HOST=ray@127.0.0.1
RAY_PORT=8000
PROFILER_ENDPOINT=http://profiler@127.0.0.1:8000
INSPECTOR_URL=http://inspector@127.0.0.1:8000
INSPECTOR_API_KEY=test
HTTP_DUMP_ENDPOINT=http://http-dump@127.0.0.1:8000
```

Any other stack: send mail to SMTP `127.0.0.1:1025`, send any HTTP request to `http://127.0.0.1:8000` to see it as a dump, and put mock rules under `http://127.0.0.1:8000/mock/`.

More setups (Spiral, WordPress, Magento, Docker Compose, Kubernetes): [Getting started](https://docs.buggregator.dev/getting-started.html).

| Port | What listens |
|------|--------------|
| `8000` | Web UI, HTTP events (Sentry, Ray, Inspector, XHProf, HTTP dumps, SMS, Xdebug traces), mock server |
| `1025` | SMTP |
| `9912` | VarDumper (not on Windows) |
| `9913` | Monolog |
| `8080` | HTTP proxy |

## What it catches

| | Send from | Docs |
|---|---|---|
| Sentry: exceptions, traces, logs, metrics | Sentry SDK (DSN) | [Sentry](https://docs.buggregator.dev/config/sentry.html) |
| Emails | any SMTP client | [SMTP](https://docs.buggregator.dev/config/smtp.html) |
| Dumps | Symfony VarDumper, Spatie Ray | [VarDumper](https://docs.buggregator.dev/config/var-dumper.html), [Ray](https://docs.buggregator.dev/config/ray.html) |
| Logs | Monolog (JSON over TCP) | [Monolog](https://docs.buggregator.dev/config/monolog.html) |
| Profiles | XHProf | [XHProf](https://docs.buggregator.dev/config/xhprof.html) |
| APM transactions | Inspector | [Inspector](https://docs.buggregator.dev/config/inspector.html) |
| Any HTTP request | curl, webhooks, your app | [HTTP dumps](https://docs.buggregator.dev/config/http-dumps.html) |
| Outgoing HTTP calls | HTTP proxy on `:8080` | [HTTP proxy](https://docs.buggregator.dev/config/http-proxy.html) |
| SMS | requests in the format of 42 SMS providers (Twilio, Vonage, Infobip…) | [SMS](https://docs.buggregator.dev/config/sms.html) |
| Xdebug function traces | `curl` upload of a trace file | [Xdebug traces](https://docs.buggregator.dev/config/xdebug-trace.html) |

And around it:

- **Mock server.** Requests to `/mock/...` get the response of the first matching rule: method + path → status, headers, body, delay. Rules from the UI, the API, `buggregator.yaml`, Go templates for dynamic bodies, or one rule per operation from an OpenAPI 3.x spec. [Docs](https://docs.buggregator.dev/config/mock-server.html)
- **Email testing API.** `GET /api/smtp/messages/wait` holds the request until the email your test expects arrives; `/links` and `/codes` pull the confirmation link or the one-time code out of it. [Docs](https://docs.buggregator.dev/config/smtp.html)
- **Xdebug without an IDE (off by default).** Buggregator sits between Xdebug and your IDE. With no IDE attached, a script pauses at `xdebug_break()` or a saved breakpoint, and you continue or step from the browser. [Docs](https://docs.buggregator.dev/config/xdebug.html)
- **Projects.** One instance for several apps: each event goes to the project named in its DSN, header or SMTP username. [Docs](https://docs.buggregator.dev/config/projects.html)
- **Webhooks, SSO, Prometheus metrics** for shared or CI setups. [Webhooks](https://docs.buggregator.dev/config/webhooks.html) · [SSO](https://docs.buggregator.dev/config/sso.html) · [Metrics](https://docs.buggregator.dev/config/metrics.html)

## For AI agents (MCP)

Buggregator has a built-in MCP server. Claude Code, Cursor or any MCP client can list events, read a Sentry error with its stack trace, read dumps and profiles, find the slow calls in an Xdebug trace, and create mock rules. With the Xdebug proxy on, the agent can also set breakpoints and step through PHP code.

```bash
MCP_ENABLED=true buggregator
claude mcp add buggregator -- buggregator mcp
```

23 tools by default; 35 with the Xdebug proxy, 36 with `allow_eval`. On Windows or in Docker, use the HTTP transport. Setup for other clients: [MCP](https://docs.buggregator.dev/config/mcp.html). Want an agent to install and set it up for you? Give it [this page](https://docs.buggregator.dev/agent-setup.html).

## Configuration

Zero config to start. Every setting is a flag, an environment variable or a line in `buggregator.yaml` (flags win over variables, variables over the file). Annotated example: [`buggregator.yaml.example`](buggregator.yaml.example). All options: [Server configuration](https://docs.buggregator.dev/config/server.html).

## Build from source

Needs Go 1.26+, PHP 8.1+ with Composer (for the VarDumper parser) and Make.

```bash
make build    # downloads the UI, builds the PHP parser, compiles the binary
make run
make build-cross GOOS=darwin GOARCH=arm64
```

## Like it?

**If Buggregator replaced a tool or two on your machine, [give it a star](https://github.com/buggregator/server/stargazers) — that is how other developers find it.**

- Website: [buggregator.dev](https://buggregator.dev)
- Docs: [docs.buggregator.dev](https://docs.buggregator.dev)
- Chat: [Discord](https://discord.gg/vDsCD3EKUB)
- Support the author: [Patreon](https://patreon.com/butschster)
- No binary or Docker? [Buggregator Trap](https://docs.buggregator.dev/trap/what-is-trap.html) is a small PHP CLI alternative.
- Buggregator 1.x (PHP) lives on the [1.x branch](https://github.com/buggregator/server/tree/1.x); new work happens here.

## Contributing

Found a bug or have an idea? Open an issue or a pull request. How to set up the project: [Contributing](https://docs.buggregator.dev/contributing.html).

## License

MIT — see [LICENSE](LICENSE).
