# Insights Plugin — Functional & Performance Reference

A practical guide to what each scorecard does, what makes it return empty, what
performance traps live behind it, and what to fix first.

---

## 1. Plugin overview

A reimplementation of the deprecated core Mattermost Insights feature (deleted
in `mattermost/mattermost@26617fcbdc`). Eight scorecards on a single page,
each driven by a SQL aggregation query, controlled by two top-level inputs:

| Control | Values | Effect |
|---|---|---|
| **Scope** | `Team` / `My` | Which dataset is aggregated |
| **Time range** | `today` / `7_day` / `28_day` | `since` cutoff applied to most aggregations |

Layer model:

```
React Card (Cards/*Card.tsx)
  → Redux action (requestTop*)
    → Client fetch (/plugins/com.mattermost.insights/api/v1/...)
      → API handler (server/api/*.go)              ← gates: license, perm, guest
        → Store query (server/store/*.go)          ← raw SQL via squirrel
          → Postgres
```

Each card has its own `useEffect([scope, scopeKey, timeRange, teamId])`, so
**every change to any control re-fetches all 8 cards in parallel**.

### Auth gates (uniform across team-scope routes)

| Gate | Where | Affects |
|---|---|---|
| `requireUser` | every route | 401 if `Mattermost-User-Id` header missing |
| `rejectGuest` | every route | 403 for guests |
| `requireProfessionalLicense` | **team-scope only** | 403 unless Professional / Enterprise / Enterprise Advanced |
| `requireTeamPermission(PermissionViewTeam)` | team-scope only | 403 if user not in team |

User-scoped (`/users/me/top/*`) is NOT license-gated, with one exception:
Boards and Playbooks queries return empty if the respective plugin's DB
tables don't exist.

### Pagination

- `per_page` default = 60, hard cap = 100 (`server/api/params.go`)
- `page` 0-indexed
- All queries return `limit = perPage+1` then check the `+1` to set
  `has_next` (`server/insights/pagination.go`)

---

## 2. Scorecard reference

Every card below documents the columns it returns, the filters that gate its
result, and what differs between **Team** and **My** scope.

### 2.1 Top Reactions

**Function:** Most-used emoji counted from `Reactions` rows.

**Team scope (`/teams/{team_id}/top/reactions`)**
- UNION ALL of two branches:
  - Private channels of the team **where the requester is a `ChannelMember`**
  - All public channels of the team (membership-free)
- Filters by `Reactions.DeleteAt = 0` AND `Reactions.CreateAt > since`
- Group/sum across both branches.

**My scope (`/users/me/top/reactions`)**
- **`team_id` query parameter is optional**, with very different behavior:
  - **Empty `team_id`**: counts every reaction the user ever made across **all teams plus DMs and GMs**. Effectively a global "your reactions everywhere."
  - **Non-empty `team_id`**: restricts to that team's channels **plus all DMs and GMs** (DMs/GMs are not team-scoped, so they always count when a team is provided).
- Always counts only reactions the **requesting user** placed (`Reactions.UserId = me`).

**Gotchas / inconsistencies:**

- The `team_id`-less mode for "My" is **not symmetric** with any Team-scope variant. It spans teams; nothing else does.
- Reactions on deleted parent posts still count as long as the `Reactions` row itself has `DeleteAt = 0`. The query never joins back to `Posts`.
- The Team query takes `userID` for the private-channel branch — so the **same team** shows different counts to two different admins if their private-channel membership differs.

---

### 2.2 Top Channels

**Function:** Channels ranked by message count in the window. Also returns a sparkline (`post_count_by_duration`) for each result.

**Team scope (`/teams/{team_id}/top/channels`)**
- UNION ALL:
  - All public channels of the team (membership-free)
  - Private channels of the team where the requester is a `ChannelMember`
- Posts filtered: `DeleteAt = 0` AND `CreateAt > since` AND `Type = ''` (no system messages) AND **integration filter** (see §3.5).
- Order: `MessageCount DESC, Name ASC`.

**My scope (`/users/me/top/channels`)**
- Channels the user is a member of (public OR private), filtered to posts where `Posts.UserId = me` (i.e. **channels the user has posted in personally**, not channels they read).
- `team_id` optional. Empty → all teams; non-empty → restricted to that team.

**Gotchas / inconsistencies:**

- "My Top Channels" requires the user to have **authored a post**, not just be a member. Lurkers see empty.
- The integration filter (`from_bot/from_webhook/from_oauth_app/from_plugin`) is applied via JSONB lookups on `Posts.Props` — not indexable.
- Returns at most 5 channels worth of chart data by default (the UI uses the top result for the line chart).
- Charting follow-up query (`PostCountsByDuration`) is a **second DB round-trip** per request.

---

### 2.3 Top Threads

**Function:** Most-active threads (by reply count) in the time window.

**Team scope (`/teams/{team_id}/top/threads`)**
- All threads in public channels of the team, plus private channels where the requester is a `ChannelMember`.
- Filtered by `t.LastReplyAt > since`.

**My scope (`/users/me/top/threads`)**
- Same channel scoping (public + private-member).
- **Additionally requires** a `ThreadMemberships` row with `Following = TRUE` for the requesting user. The `LEFT JOIN` is gated by a `WHERE tm.UserId = $userID`, which makes it effectively an inner join.

**Gotchas / inconsistencies — biggest gotcha pile on the page:**

| Symptom | Cause |
|---|---|
| Thread visible in Team Top Threads but absent from My Top Threads, even when user is OP | No `ThreadMemberships` row exists for them. Auto-follow only writes the row when CRT is on AND a reply lands. Older/imported/load-tested data won't have rows. |
| Empty across the board on a fresh test server | `ThreadMemberships` table is empty because `/test posts`-style data doesn't go through the reply API path. Backfill with the SQL in §6.2 of this doc. |
| User unfollowed a thread → still owns it | `Following = false` excludes it from My, even if they're the root author. |
| Returns thread, but root post is `null` | Root post deleted; `hydrateThreads` calls `GetPost(item.PostID)` for every item, and a deleted post returns nil. |

- The hydration step is an **N+1**: one `GetPost` per thread, up to 60 round trips per request. See §3.1.

---

### 2.4 Top DMs (user-only)

**Function:** Your most-active DM partners by message count.

**Route:** `/users/me/top/dms` — **no team scope, no Team variant**. Always user.

**Query shape:**
1. Inner subquery picks DM channels the user is a member of, excluding self-DMs (`Name = '<id>__<id>'`) and DMs with bot participants.
2. Outer aggregation counts posts since cutoff.
3. Final filter drops "solo DMs" (the other participant left the workspace) via `POSITION(',' IN Participants) > 0`.

**Gotchas / inconsistencies:**

- **MessageCount is doubled in the SQL.** Because DM channels have 2 members and the join multiplies rows, the store returns `MessageCount = real_count × 2`. The handler **divides by 2** in `hydrateTopDMs` (`server/api/dms.go:58`). Anyone reading the store result directly will see inflated counts.
- A **third DB round-trip** runs after hydration: `OutgoingDMCounts` to compute "how many of these posts did I send" for the UI's outgoing-message ratio bar.
- Time filter uses `p.UpdateAt > since`, not `p.CreateAt > since`, **only on this insight**. Means edited posts re-qualify into a newer window. Inconsistent with every other insight.
- Bot exclusion uses `SPLIT_PART(Channels.Name, '__', N) NOT IN (SELECT UserId FROM Bots)`. Not indexable.

---

### 2.5 Top Inactive Channels ("Least Active Channels")

**Function:** Channels in the team / your membership set ordered by message count **ascending** — least active first.

**Team scope:** public channels of the team + private channels you're a member of.
**My scope:** public OR private channels you're a member of. `team_id` optional.

**The killer filter:** `Channels.CreateAt < $since`.

A channel only qualifies if it was created **before the start of the time
window**. The reasoning: a 3-day-old channel can't fairly be called "inactive
over the last 28 days." But on fresh dev servers this rules out every channel
for any non-trivial time range.

**Gotchas / inconsistencies:**

| Symptom | Cause | Fix |
|---|---|---|
| Empty on fresh dev server | Every channel's `CreateAt > since` | Pick **shorter** time range (Today), or backdate channels (see §6.3) |
| Channel with zero posts appears at top | `LEFT JOIN Posts` returns the channel with `MessageCount = 0` | Working as designed — that *is* "least active" |
| Private channel missing from "My" but present in Team for the same user | User must be a `ChannelMember`. Make sure the join row exists |
| Separate `attachInactiveChannelParticipants` query | Always a second DB round-trip per request, even if zero results — actually no, it short-circuits on `len == 0` |

---

### 2.6 New Team Members (team-only)

**Function:** Users who joined the team since the start of the window, with a total count and paginated list. Most-recent-first.

**Route:** `/teams/{team_id}/top/team_members` — **no My variant.**

**Filter:**
- `TeamMembers.CreateAt >= since` (note: `>=`, **strict-inclusive**, different from every other insight which uses `>`).
- `TeamMembers.DeleteAt = 0`, `Users.DeleteAt = 0`, `Bots.UserId IS NULL` (excludes bots).

**Privacy gate:** `users.firstname` / `lastname` populated only when **either** `PrivacySettings.ShowFullName = true` **or** the requester is a system admin. Both `Username` and `Nickname` are always returned.

**Gotchas / inconsistencies:**

- The `>=` comparison is the **only** insight using inclusive-since. Every other store query uses strict `>`. Practically irrelevant (millisecond resolution), but worth knowing during debugging.
- The handler does **two queries**: one `count(*)` then one paginated `SELECT`. Same `WHERE` clause executes twice. See §3.2.
- Admin override on privacy is server-side; the webapp doesn't know whether full names are real or blanked. UI just renders what arrives.

---

### 2.7 Top Boards

**Function:** Most-active boards (boards + cards combined edit count) over the window.

**Route:** `/teams/{team_id}/top/boards` (Team) or `/users/me/top/boards` (My).

**Data source:** `focalboard_boards_history` + `focalboard_blocks_history` — tables owned by the **Boards plugin**, not Mattermost core.

**ACL:** API handler first calls `BoardIDsForUserInTeam` to compute the user's accessible board set; the activity aggregation only counts boards in that set.

**Filters:** `modified_by != 'system'` (no auto-edits), `boards.delete_at = 0`.

**Gotchas / inconsistencies:**

- **Silent empty when Boards plugin not installed.** The `focalboard_*` tables don't exist, the query errors at the Postgres level, and the handler surfaces 500. The webapp shows the generic error card. **There is no precheck.**
- Even with Boards installed, a board with zero block edits but several board-config edits will rank lower than expected — the UNION counts them differently than users intuit.
- ACL computation (`BoardIDsForUserInTeam`) is itself a Postgres query that runs **before** the activity query — so this card is a **2-query handler** like Top Channels.

---

### 2.8 Top Playbooks

**Function:** Most-run playbooks (Playbooks plugin) over the window.

**Route:** `/teams/{team_id}/top/playbooks` or `/users/me/top/playbooks`.

**Data source:** `IR_Playbook`, `IR_Incident`, `IR_PlaybookMember` — tables owned by the **Playbooks plugin**.

**License note (from `server/api/api.go:41`):** this plugin owns its own license gate, which accepts Enterprise **Advanced**. Querying via the Playbooks plugin directly would use Playbooks's gate, which doesn't.

**Gotchas / inconsistencies:**

- **Silent empty when Playbooks plugin not installed**, same shape as Boards.
- Plugin coupling: a schema change in the Playbooks plugin (renaming `IR_Incident` → something) silently breaks this card.

---

## 3. Performance issues & expensive SQL patterns

### 3.1 N+1 in `hydrateThreads`

```go
for _, item := range list.Items {
    post, postErr := a.directory.GetPost(item.PostID)  // 🚨 one call per thread
    ...
}
```

`server/api/threads.go:106`. With `per_page = 60` default, this is up to **60 sequential `GetPost` calls** per Top Threads request. The user lookup right above it is correctly batched (`ListUsersByIDs`), so this is purely a port miss.

**Cost:** each `GetPost` is a primary-key lookup so ~1ms in best case, but the round-trip overhead through `pluginapi` is the real bill. Easily 50–200ms added latency per request.

### 3.2 Duplicate WHERE clauses in New Team Members

```go
countSQL := newTeamMembersSelect(... "count(*)").ToSql()
listSQL  := newTeamMembersSelect(... full columns ...).ToSql()
```

Same multi-join `WHERE` runs twice. For a count, `EXPLAIN` will use the index; for the paginated `SELECT`, ditto. But it's two round trips and two parses. A single `SELECT COUNT(*) OVER ()` window function would collapse it into one query.

### 3.3 PostCountsByDuration — non-indexable GROUP BY

```sql
GROUP BY TO_CHAR(TO_TIMESTAMP(Posts.CreateAt / 1000) AT TIME ZONE 'UTC', 'YYYY-MM-DD')
```

Function-on-column GROUP BY. Postgres evaluates `TO_CHAR(TO_TIMESTAMP(...) AT TIME ZONE ...)` for every Post row in the time window. No index can help. For `28_day` on a large server, this scans **every Post in the last 28 days** for every Top-Channels request.

### 3.4 Top DMs — multiple subqueries with non-indexable predicates

```sql
SPLIT_PART(Channels.Name, '__', 1) NOT IN (SELECT UserId FROM Bots)
SPLIT_PART(Channels.Name, '__', 2) NOT IN (SELECT UserId FROM Bots)
```

Two function-call NOT IN subselects per DM channel evaluated. Solo-DM filter (`POSITION(',' IN Participants) > 0`) runs **after** aggregation — so DMs that will be filtered out are still scanned and aggregated. Inefficient.

### 3.5 Integration filter on Posts.Props (JSONB)

Every channel/thread/chart query carries:

```sql
AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
AND (Posts.Props ->> 'from_webhook' IS NULL OR ...) -- and so on for 4 keys
```

JSONB `->> ` extraction on every scanned Post row. Not indexed by default. On large Posts tables this dominates the cost of the filter portion. The original Mattermost server pays the same price.

A partial GIN or expression index on the affected keys would help, but altering core Mattermost tables from a plugin is not viable. The realistic mitigation is caching.

### 3.6 Top Channels for Team — UNION ALL of two full Posts scans

The team query is two Posts aggregations stitched with UNION ALL — one for public channels, one for private. Each scans Posts with `CreateAt > since` and filters by team via a join. On installs with millions of recent Posts, this is the most expensive query on the page.

### 3.7 Top Inactive Channels for Team — full channel scan

`LEFT JOIN Posts` from every public channel in the team. Output row count = team's channel count (every channel appears, even with zero posts). On teams with thousands of channels, the GROUP BY across all of them is expensive even though the per-row work is small.

### 3.8 Frontend fan-out

`InsightsPage.tsx` mounts 8 cards in parallel. Each `useEffect` re-fires on
**any** of: scope toggle, time-range change, team switch, page mount.

- Single page open: **8 requests** in flight.
- Top Channels endpoint: **+1 extra DB query** (PostCountsByDuration).
- Top Threads endpoint: **+ N DB queries** for hydration (up to 60).
- Top DMs endpoint: **+1 extra DB query** (OutgoingDMCounts).
- New Team Members: **+1 extra DB query** (count).
- Top Boards / Top Playbooks: **+1 ACL query** each.

A single Insights-page load can issue **15–80 DB round trips**. Multiplied
by users with no caching: every page open is a fresh fan-out.

### 3.9 No caching, no coalescing, no rate limiting

- Zero in-memory cache in the plugin.
- No Redis or shared cache.
- No HTTP cache headers (`Cache-Control`, `ETag`).
- No `singleflight` — 100 users hitting the same endpoint simultaneously = 100 identical queries.
- No rate limiting at the plugin layer — a user holding the time-range hotkey will issue a request per render.

### 3.10 Other rough edges

- **No SQL timeouts.** Queries inherit only the HTTP context; a slow query holds a connection until the client gives up.
- **No frontend debounce** on the controls. Rapid clicks fan out fully.
- **Redux slices are cached by `{scope, scopeKey, timeRange}`**, but the action dispatches a re-fetch every time the effect's deps change, so the cache prevents flicker but not network traffic.

---

## 4. Practical solutions (prioritized 80/20)

### P0 — Single-week, biggest leverage

1. **In-plugin response cache** (60–300s TTL), keyed by `{handler, userID/teamID, timeRange, page, perPage}`. One change, eliminates most repeat traffic. Recommended: `golang-lru/v2` with a per-handler TTL.
2. **Batch `hydrateThreads`** — add `GetPostsByIDs([]string)` to the `Directory` interface, implement it via `pluginapi.Post.GetPostsByIDs` or a single `Posts WHERE Id IN (...)` query. Removes the only true N+1.
3. **Add `singleflight` to hot endpoints** so concurrent identical requests share one query. Particularly cheap to add in front of the per-handler entry.
4. **Hard-cap `per_page`** further down (e.g. 25 for cards, 100 for an explicit "details" endpoint). Today's cap of 100 lets a misbehaving client request 100 thread hydrations.
5. **SQL timeouts** — add `context.WithTimeout(ctx, 10*time.Second)` in each handler before calling the store.

### P1 — Second-tier wins

6. **Collapse New Team Members into a single query** using `COUNT(*) OVER ()`.
7. **Debounce frontend controls** (250ms). Trivial; users can't perceive it.
8. **Make user-scope time filters use `CreateAt` not `UpdateAt` in Top DMs** for consistency with the rest of the page.
9. **Precheck Boards / Playbooks plugin presence** before running their queries; return a structured `not_available` response instead of a 500.

### P2 — Deployment / ops

10. **Audit Postgres indexes on the deployment target.** Confirm `(Posts.ChannelId, Posts.CreateAt)`, `(Reactions.UserId, Reactions.CreateAt)`, `(Threads.LastReplyAt)`, `(ThreadMemberships.UserId, ThreadMemberships.PostId)`, `(TeamMembers.TeamId, TeamMembers.CreateAt)` are present.
11. **Consider an expression / partial index** on `Posts.Props ->> 'from_bot'` etc. — biggest payoff if integration traffic is significant.
12. **Materialize `PostCountsByDuration` daily** for the `28_day` range — a nightly job that pre-computes per-channel per-day post counts into a small table. Turns the sparkline query into a constant-cost lookup.

### P3 — Architectural

13. **Move heavy aggregations off the request path** via a `mattermost-plugin-insights-precompute` job (cron-style hook) that builds materialized tables. Most insights data changes on a daily cadence, not per-request.
14. **Replace per-request DB hits with a snapshot model**: every 5 minutes, run all queries once for each `(team, timeRange)` and store the JSON; serve the cached JSON to all users on hit. Massively simpler than per-user caching when teams are the cardinality bound.

---

## 5. Cross-cutting concerns that don't fit in one card

### 5.1 Time-range semantics

All cards use **start-of-day in the user's timezone**, not "now minus 24h." So
"Today" means since-midnight; "7_day" means since-midnight-7-days-ago. Driven
by `StartOfDayForTimeRange` in `server/insights/timerange.go`. Affected by:

- Server time zone? No — driven by `user.GetTimezoneLocation()`.
- DST transitions? Yes — a 7-day window straddling a DST change is 167 or 169 hours, not 168. Inherited from original.

### 5.2 Bot / integration exclusion is inconsistent

- **Posts-scoped queries** (Top Channels, Top Threads via thread root, charting) apply the four `Posts.Props ->> 'from_X'` filters.
- **Reactions** queries do NOT filter integration-authored reactions. A bot that fires reactions will skew Top Reactions. (Mattermost bots rarely add reactions, so this is theoretical, but worth noting.)
- **New Team Members** correctly excludes bots via `Bots.UserId IS NULL`.
- **Top DMs** excludes bot-participant DMs via SPLIT_PART, but only excludes when the bot is at the start of the DM name. (Practically OK because MM sorts the two user IDs in the DM channel name, so a bot's ID lands in one position or the other deterministically.)

### 5.3 Privacy

`ShowFullName` is honored **only** in New Team Members. Top Reactions, Top
Channels, Top Threads, etc. include `User.FirstName` / `LastName` regardless.
This matches the original Mattermost behavior but means privacy enforcement
is inconsistent across the page.

### 5.4 Soft-delete behavior

- `Posts.DeleteAt = 0` enforced on every Posts query.
- `Channels.DeleteAt = 0` enforced on every Channels query.
- `Threads.threaddeleteat IS NULL` enforced on Threads queries (note: `IS NULL`, not `= 0`).
- `Reactions.DeleteAt = 0` enforced.
- `TeamMembers.DeleteAt = 0`, `Users.DeleteAt = 0` enforced on New Team Members.

The reaction query specifically uses the **inner** soft-delete inside the
UNION ALL (so the GROUP BY preserves DeleteAt for the outer filter) — that
unusual `GROUP BY EmojiName, DeleteAt, CreateAt` shape is verbatim from the
original Mattermost code and looks redundant but is structurally required
by the UNION-ALL post-filter.

### 5.5 Test-data behavior

| Data source | ThreadMemberships | Reactions.UserId | TeamMembers | Notes |
|---|---|---|---|---|
| Real UI activity | populated | populated | populated | Everything works |
| `/test posts` / `/test url` | **not populated** | populated | populated | My Top Threads silently empty |
| Bulk import (`mmctl import`) | depends on importer | populated | populated | Verify after import |
| Direct SQL `INSERT INTO Posts` | empty | empty | empty | Test fixture only |

If sample data has missing `ThreadMemberships`, see §6.2 for a backfill.

### 5.6 Plugin coupling risk

- Top Boards depends on `focalboard_*` tables. Versioning of those tables is governed by the Boards plugin, not this plugin. A Boards plugin schema migration could silently break this card.
- Top Playbooks depends on `IR_*` tables, same deal.
- Both cards swallow plugin-absence gracefully only if the queries fail at a level the handler recognizes; today they likely return 500. Worth adding a precheck.

### 5.7 Telemetry

`server/api/api.go` wires a `Telemetry` interface. Production implementation
writes structured-log events via `pluginapi`. No PII; admins can scrape and
forward to their own pipeline. Worth flagging that this is **fire-and-forget**:
no batching, no flush guarantees, no rate limiting on the events.

---

## 6. Reference SQL / quick fixes

### 6.1 Sanity check: which cards have data right now

```sql
-- Posts in last 28 days
SELECT count(*) FROM Posts
WHERE DeleteAt = 0 AND CreateAt > (EXTRACT(EPOCH FROM NOW() - INTERVAL '28 days') * 1000)::bigint;

-- Reactions in last 28 days
SELECT count(*) FROM Reactions
WHERE DeleteAt = 0 AND CreateAt > (EXTRACT(EPOCH FROM NOW() - INTERVAL '28 days') * 1000)::bigint;

-- ThreadMemberships at all
SELECT count(*) FROM ThreadMemberships WHERE Following = TRUE;

-- New team members in last 28 days (replace team_id)
SELECT count(*) FROM TeamMembers
WHERE DeleteAt = 0 AND TeamId = '<team_id>'
  AND CreateAt >= (EXTRACT(EPOCH FROM NOW() - INTERVAL '28 days') * 1000)::bigint;

-- Channels old enough to qualify for "Least Active" in any window
SELECT count(*) FROM Channels
WHERE DeleteAt = 0 AND Type IN ('O','P')
  AND CreateAt < (EXTRACT(EPOCH FROM NOW()) * 1000)::bigint;
```

### 6.2 Backfill `ThreadMemberships` from existing threads

Adds a `Following = TRUE` row for each thread's original author. Re-running is safe (skips existing rows).

```sql
INSERT INTO ThreadMemberships (PostId, UserId, Following, LastUpdated, LastViewed, UnreadMentions)
SELECT
    t.PostId,
    p.UserId,
    TRUE,
    (EXTRACT(EPOCH FROM NOW()) * 1000)::bigint,
    0,
    0
FROM Threads t
JOIN Posts p ON p.Id = t.PostId
WHERE NOT EXISTS (
    SELECT 1 FROM ThreadMemberships tm
    WHERE tm.PostId = t.PostId AND tm.UserId = p.UserId
);
```

Broader version that also adds repliers:

```sql
INSERT INTO ThreadMemberships (PostId, UserId, Following, LastUpdated, LastViewed, UnreadMentions)
SELECT DISTINCT t.PostId, p.UserId, TRUE,
       (EXTRACT(EPOCH FROM NOW()) * 1000)::bigint, 0, 0
FROM Threads t
JOIN Posts p ON p.RootId = t.PostId
WHERE NOT EXISTS (
    SELECT 1 FROM ThreadMemberships tm
    WHERE tm.PostId = t.PostId AND tm.UserId = p.UserId
);
```

### 6.3 Backdate channels so "Least Active" returns data

```sql
-- Make every channel look 30 days old
UPDATE Channels SET CreateAt = CreateAt - (30::bigint * 24 * 3600 * 1000);
```

### 6.4 Quick endpoint poke for debugging

```sh
# Replace HOST + token with your local server
curl -s -H "Authorization: Bearer <token>" \
  "http://localhost:8065/plugins/insights/api/v1/users/me/top/threads?team_id=<team>&time_range=28_day" | jq .
```

---

## 7. Open questions / unfinished business

- No structured way to surface "Boards / Playbooks not installed" — today it likely 500s.
- Hardcoded `Posts.Props ->> 'from_*'` integration filter — should this match what core MM does today? Worth diffing against current `mattermost/mattermost` server.
- No load testing on file. The §3 cost analysis is structural reasoning, not measured numbers.
- No frontend perf budget (initial mount of 8 cards in parallel might still be acceptable depending on backend caching).
