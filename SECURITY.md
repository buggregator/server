# Security policy

## Supported versions

Security fixes go into the newest release only. To get a fix, upgrade to the newest release.

| Version | Gets security fixes |
|---------|---------------------|
| newest 2.x release | yes |
| older 2.x releases | no, upgrade |
| 1.x (PHP version) | no, no longer maintained |

## Reporting a vulnerability

Please do not open a public issue.

Report it privately on GitHub: open the [Security tab](https://github.com/buggregator/server/security) of this
repository and click **Report a vulnerability**. Only the maintainers see the report.

Please include:

- the Buggregator version (`GET /api/version` or the release you run) and how you run it (binary, Docker, Helm);
- what an attacker can do and what they need for it (e.g. network access to a port);
- steps or a request that shows it.

## Before you report

Buggregator is a debugging tool for a developer machine or a trusted dev network. Some behaviour is by design and
documented in [Security and data](https://docs.buggregator.dev/security): for example, ports listen on all
interfaces by default, and event ingestion stays open when SSO is on. A way to get past what that page says is
a vulnerability; what the page already describes is not.
