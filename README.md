# Mattermost Insights Plugin

Surfaces activity analytics inside Mattermost: top channels, top reactions, top threads, top DMs, top inactive channels, and new team members. Each insight is available across **today**, **7-day**, and **28-day** windows, scoped either to a whole team or to your personal activity. The Insights surface mounts as a top-level product in the team switcher.

## Insights at a glance

| Insight | Team scope | My scope | Notes |
| --- | --- | --- | --- |
| Top Channels | ✅ | ✅ | Most active channels by message count, plus a per-bucket post-count chart (hourly for "today", daily otherwise) |
| Top Reactions | ✅ | ✅ | Most-used emoji reactions |
| Top Threads | ✅ | ✅ | Most active threads, with channel name, root-post author, and participants |
| Top DMs | — | ✅ | Direct-message partners by message count, with outgoing share and partner profile |
| Top Inactive Channels | ✅ | ✅ | Channels with the least activity, with participant ids |
| New Team Members | ✅ | — | Members who joined in the selected window, with `total_count` for pagination |

## Requirements

- Mattermost Server 11.3.0 or newer
- PostgreSQL — Mattermost has dropped MySQL support, so the plugin targets PostgreSQL exclusively
- Mattermost **Professional**, **Enterprise**, or **Enterprise Advanced** license — required for the team-scoped insights; my-scope insights work on any license

## Install

1. Download the plugin bundle from the latest release.
2. In Mattermost: **System Console → Plugin Management → Upload Plugin**, select the bundle, and enable.
3. Open any team — an "Insights" entry appears in the team product switcher.

## REST routes

All routes are mounted under `/plugins/insights/api/v1/`. Common query params: `time_range` (`today`, `7_day`, `28_day`; required), `page` (default 0), `per_page` (default 60, capped at 100).

| Method | Path | Auth |
| --- | --- | --- |
| GET | `/teams/{team_id}/top/reactions` | Professional/Enterprise/Enterprise Advanced, view_team |
| GET | `/teams/{team_id}/top/channels` | Professional/Enterprise/Enterprise Advanced, view_team |
| GET | `/teams/{team_id}/top/threads` | Professional/Enterprise/Enterprise Advanced, view_team |
| GET | `/teams/{team_id}/top/inactive_channels` | Professional/Enterprise/Enterprise Advanced, view_team |
| GET | `/teams/{team_id}/top/team_members` | Professional/Enterprise/Enterprise Advanced, view_team |
| GET | `/users/me/top/reactions` | self (optional `team_id` filter) |
| GET | `/users/me/top/channels` | self (optional `team_id` filter) |
| GET | `/users/me/top/threads` | self (optional `team_id` filter) |
| GET | `/users/me/top/dms` | self |
| GET | `/users/me/top/inactive_channels` | self (optional `team_id` filter) |

Guests are rejected from every route.

## Build and deploy

- `make dist` — build the plugin bundle
- `make check-style` — lint Go and webapp code
- `make test` — run server tests (testcontainers Postgres) + webapp Playwright tests
- `make docker-setup` — first-time setup of the local Mattermost dev environment
- `make deploy` — deploy to the Mattermost server in `docker-compose.dev.yml`

The server tests require Docker (testcontainers boots a Postgres container per `go test` invocation). On macOS, OrbStack or Docker Desktop both work.

## Security scanning

- `make sbom` — generate CycloneDX SBOMs for the Go server and npm webapp into `dist/sbom/`
- `make sbom-scan` — run grype against the SBOMs and fail on high or critical CVEs
- `make sbom-audit` — both in one step
- `make codeql-analyze` — run GitHub CodeQL queries on the Go and JavaScript/TypeScript sources, writing SARIF to `dist/codeql-*.sarif`
- `make security-gate` — check the SARIF files for critical/high findings and exit non-zero if any are present

Both grype and CodeQL are clean as of v0.1.0. Suppress known false-positive grype findings via `.grype.yaml`.

See the [Mattermost plugin developer docs](https://developers.mattermost.com/extend/plugins/) for general plugin guidance.
