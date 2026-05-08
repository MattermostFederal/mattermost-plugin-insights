# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-05-07

### Added
- Insights product page mounted via `registerProduct`, with My / Team scope
  selector, time-range selector (Today / 7 day / 28 day), and a six-card
  dashboard.
- Top Reactions (team and my scope) — most-used emoji reactions.
- Top Channels (team and my scope) — most active channels by message count,
  with a per-bucket post-count sparkline (hourly for Today, daily otherwise).
- Top Threads (team and my scope) — most active threads, with channel name,
  root-post author profile, and participants.
- Top DMs (my scope) — direct-message partners by message count, with
  partner profile and outgoing-message share.
- Top Inactive Channels (team and my scope) — channels with the least
  activity in the window.
- New Team Members (team scope) — members who joined the team in the window,
  honoring `PrivacySettings.ShowFullName` (admins always see full names).
- License & permission gating: team-scoped routes require a Mattermost
  Professional, Enterprise, or Enterprise Advanced license plus the
  `view_team` permission; guests are rejected from every route; my-scope
  routes work on any license.

### Compatibility
- Requires Mattermost Server 11.3.0 or newer.
- PostgreSQL only. MySQL is not supported (Mattermost itself has dropped
  MySQL support).

### Security
- `make sbom-audit` (CycloneDX SBOMs + grype CVE scan) reports no
  vulnerabilities in Go or webapp dependencies.
- `make codeql-analyze` + `make security-gate` (GitHub CodeQL on Go and
  JavaScript/TypeScript) report no critical or high findings.
