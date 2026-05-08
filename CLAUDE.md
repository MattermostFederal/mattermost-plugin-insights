# Mattermost Insights Plugin Guidelines

## Overview

A plugin reimplementation of the deprecated Mattermost Insights feature (deleted in mattermost/mattermost commit `26617fcbdc`). The server is Go; the webapp is TypeScript / React / Redux. Plan and progress live in [PLAN.md](PLAN.md).

## Architecture

```
server/
  main.go                  – plugin.ClientMain entry
  plugin.go                – Plugin struct (embeds plugin.MattermostPlugin),
                             OnActivate wires pluginapi.Client + Store + API,
                             ServeHTTP delegates to api.API
  configuration.go         – minimal; no insights-specific settings yet
  command.go               – /insights help slash command
  api/
    api.go                 – gorilla/mux router; defines Storer / AuthProvider /
                             Directory interfaces; FromPluginAPI adapter wires
                             pluginapi.Client into all three
    auth.go                – requireUser / rejectGuest /
                             requireProfessionalLicense (accepts Professional,
                             Enterprise, and Enterprise Advanced) /
                             requireTeamPermission
    {reactions, threads, channels, inactive_channels, dms, team_members}.go
                           – per-insight HTTP handlers + post-processing
    apitest/stubs.go       – AuthStub / DirectoryStub / StoreStub for fast,
                             Docker-free handler tests
  insights/
    types.go               – domain types (TopReaction, TopChannel, ...)
    timerange.go           – today / 7_day / 28_day → start unix-ms
    pagination.go          – limit+1 → has_next slicing
    post_count_by_duration.go
                           – view-model assembly for the chart
  store/
    store.go               – pluginapi-backed *sql.DB + squirrel builder
    {reaction, thread, channel, inactive_channel, dm, team,
     post_count_by_duration}.go
                           – raw-SQL aggregation queries (PostgreSQL)
    storetest/
      schema.sql           – minimal Mattermost-shaped schema
      harness.go           – session-scoped testcontainers Postgres

webapp/src/
  index.tsx                – registerProduct + reducer + i18n
  client/Client.ts         – fetch wrapper for /plugins/insights/api/v1/...
  redux/{actionTypes, actions, reducer, selectors, types}.ts
                           – per-insight (LOADING|RECEIVE|ERROR) triplets and
                             a normalized {scope: {scopeKey: {timeRange: slice}}}
                             state shape
  components/
    Page/InsightsPage.tsx  – top-level page; reads scope+timeRange from URL
                             query params
    Controls/{ScopeSelect,TimeRangeSelect}.tsx
    Card/InsightCard.tsx   – generic card shell
    Cards/{TopReactions,TopChannels,TopThreads,TopDMs,TopInactiveChannels,
           NewTeamMembers}{List,Card}.tsx
                           – split presentational List from Redux-bound Card
    Charts/PostCountChart.tsx
                           – tiny SVG sparkline for Top Channels
```

## Coding conventions

- **Match surrounding code.** When in doubt, mirror the closest existing pattern.
- **Server logging:** use `p.API.LogError` / `LogWarn` / `LogInfo`.
- **Webapp:** prefer functional React components with hooks. Cards split presentational `*List` (no Redux) from Redux-bound container `*Card` so the dumb component is trivially testable in Playwright CT.
- **SQL queries are squirrel-built.** When porting a query from the deleted Mattermost server source, read the original verbatim via `git -C ../mattermost show 26617fcbdc -- <path>` and adapt placeholder format with `sq.Dollar` (Postgres). The team UNION-ALL queries' grouping shape is preserved verbatim, including the unusual `GROUP BY ... t.DeleteAt, t.CreateAt` patterns that look redundant — Postgres accepts them because the primary key is in the GROUP BY, and removing them would change the soft-delete / time-window behavior.

## Adding a new insight (TDD)

The codebase enforces a strict test-first cycle, encoded by feedback memory and demonstrated for every existing insight:

1. **Store layer (testcontainers Postgres).** Recover the original SQL with `git show 26617fcbdc -- <store path>`. Write `server/store/<insight>_test.go` with table-driven cases against fixtures inserted directly into the test schema. Stub the store method to return `errStoreNotImplemented`, run `go test` → red, then port the SQL → green.
2. **API layer (httptest + stubs).** Write `server/api/<insight>_test.go` against `apitest.StoreStub` / `apitest.DirectoryStub` / `apitest.AuthStub`. Cover the gate matrix (`gates_test.go` enforces uniformity across routes; per-insight tests cover happy paths and any insight-specific behavior such as hydration or count fixups). Stub the handler to return 501, run → red, implement → green.
3. **Webapp layer (Playwright CT).** Write `webapp/src/components/Cards/<Insight>List.pw.tsx` covering empty/loading/error/populated states. Stub the component to return `null`, run → red, implement → green. Then build the Redux-bound `<Insight>Card.tsx` and wire it into `InsightsPage.tsx`.

The interface refactor in `server/api/api.go` (Storer + AuthProvider + Directory) is what makes the API tests fast — they run in milliseconds without Docker.

## Recovering the deleted source

The original Insights code lived in `mattermost/mattermost`; deletion commit is `26617fcbdc`. From `../mattermost`, recover any deleted file with:

```sh
git show 26617fcbdc -- server/public/model/insights.go
git show 26617fcbdc -- server/channels/api4/insights.go
git show 26617fcbdc -- server/channels/store/sqlstore/{reaction,channel,thread,post,team}_store.go
git show 26617fcbdc -- server/channels/app/{post,channel,reaction,team}.go
git show 26617fcbdc -- server/i18n/en.json
```

## Build and test

- `make dist` — build the plugin bundle
- `make check-style` — lint Go and webapp code
- `make test` — server tests + webapp Playwright tests (server tests require Docker)
- `make test-ci` — same, with JUnit XML output for CI
- `make deploy` — deploy to the Mattermost server in `docker-compose.dev.yml`
