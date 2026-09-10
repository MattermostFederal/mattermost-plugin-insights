-- Compare the two shapes of the channel-activity query against a real
-- Mattermost database.
--
-- The Go benchmark (server/store/channel_activity_bench_test.go) measures
-- synthetic data. This measures yours: real row counts, real Props
-- distribution, real data skew, real table bloat. Where the two disagree,
-- believe this one.
--
-- SAFETY: every statement here is a read-only SELECT. Nothing is written,
-- and no schema is touched. But EXPLAIN ANALYZE *executes* the query, and
-- the legacy shape is the slow one — on a large server it can run for
-- minutes and will occupy a CPU while it does. Run it against a local
-- instance, a staging copy, or a read replica. Not a production primary in
-- business hours.
--
-- Usage:
--
--   psql "$DATABASE_URL" -f scripts/channel_activity_compare.sql
--
--   # or pin a specific team instead of letting it pick the biggest
--   psql "$DATABASE_URL" -v team_id=abc123... -f scripts/channel_activity_compare.sql
--
-- For a local dev instance started from docker-compose.dev.yml, that is
-- typically:
--
--   psql "postgres://mmuser:mostest@localhost:5432/mattermost_test" \
--        -f scripts/channel_activity_compare.sql

\timing on
\pset pager off

-- ---------------------------------------------------------------------------
-- Parameters
-- ---------------------------------------------------------------------------

-- Default team_id to empty if it was not passed with -v, so the COALESCE
-- below can fall back to auto-detection.
\if :{?team_id}
\else
  \set team_id ''
\endif

-- Pick the team with the most channels unless one was named. That is the
-- worst case on this server, which is the interesting one.
SELECT COALESCE(
  NULLIF(:'team_id', ''),
  (SELECT teamid
     FROM channels
    WHERE deleteat = 0 AND type IN ('O', 'P') AND teamid <> ''
    GROUP BY teamid
    ORDER BY count(*) DESC
    LIMIT 1)
) AS team_id \gset

-- A closed 28-day window ending at midnight UTC, matching what the plugin
-- actually queries (insights.WindowUTC). Closed windows are why one snapshot
-- can serve a whole day.
SELECT
  (EXTRACT(EPOCH FROM (date_trunc('day', now() AT TIME ZONE 'UTC') - INTERVAL '28 days')) * 1000)::bigint AS window_start,
  (EXTRACT(EPOCH FROM  date_trunc('day', now() AT TIME ZONE 'UTC'))                       * 1000)::bigint AS window_end
\gset

-- The last-post lookback the query scans, two years back from the window
-- start (store.lastPostLookbackMillis).
SELECT (:window_start - (2::bigint * 365 * 24 * 60 * 60 * 1000)) AS lookback_start \gset

\echo ''
\echo '=== Parameters ==='
SELECT
  :'team_id'                                   AS team_id,
  to_timestamp(:window_start   / 1000) AT TIME ZONE 'UTC' AS window_start,
  to_timestamp(:window_end     / 1000) AT TIME ZONE 'UTC' AS window_end,
  to_timestamp(:lookback_start / 1000) AT TIME ZONE 'UTC' AS lookback_start;

-- ---------------------------------------------------------------------------
-- Dataset shape — always report this alongside any timing. A ratio without
-- the dataset it was measured on is not a result.
-- ---------------------------------------------------------------------------

\echo ''
\echo '=== Dataset ==='
SELECT
  (SELECT count(*) FROM channels
    WHERE teamid = :'team_id' AND deleteat = 0 AND type IN ('O','P'))         AS team_channels,
  (SELECT count(*) FROM channels WHERE deleteat = 0)                          AS all_channels,
  (SELECT count(*) FROM posts)                                                AS all_posts,
  (SELECT count(*) FROM posts WHERE createat >= :window_start
                                AND createat <  :window_end)                  AS posts_in_window,
  (SELECT count(*) FROM posts WHERE createat >= :lookback_start
                                AND createat <  :window_end)                  AS posts_in_lookback,
  pg_size_pretty(pg_total_relation_size('posts'))                             AS posts_size;

-- The share of posts carrying an integration prop. These four predicates are
-- the ones the planner cannot estimate through; if this is ~0 on your server
-- the misestimate may not reproduce at all, and that is a finding worth
-- knowing before quoting a number.
\echo ''
\echo '=== Integration traffic (the unestimable predicates) ==='
SELECT
  count(*) FILTER (WHERE props ->> 'from_bot'       = 'true') AS from_bot,
  count(*) FILTER (WHERE props ->> 'from_webhook'   = 'true') AS from_webhook,
  count(*) FILTER (WHERE props ->> 'from_oauth_app' = 'true') AS from_oauth_app,
  count(*) FILTER (WHERE props ->> 'from_plugin'    = 'true') AS from_plugin,
  count(*)                                                    AS total
FROM posts
WHERE createat >= :lookback_start AND createat < :window_end;

-- ---------------------------------------------------------------------------
-- Timed runs. Output is discarded so the numbers measure the query, not psql
-- formatting a few hundred rows to the terminal. Read the "Time:" line that
-- \timing prints after each.
--
-- Run the file more than once if you care about warm-cache numbers: the first
-- execution pays for reading pages off disk.
-- ---------------------------------------------------------------------------

\echo ''
\echo '=== TIMED: legacy (row-by-row LEFT JOIN, pre-d4d8604) ==='
\o /dev/null
SELECT
	Channels.Id,
	Channels.Type,
	Channels.DisplayName,
	Channels.Name,
	Channels.Purpose,
	Channels.Header,
	Channels.CreateAt,
	COALESCE(LastReal.LastRealPostAt, 0) AS LastPostAt,
	COALESCE(max(Posts.CreateAt), 0) AS LastPostInWindow,
	count(Posts.Id) AS MessageCount,
	count(DISTINCT Posts.UserId) AS ActivePosters,
	COALESCE(Members.MemberCount, 0) AS MemberCount
FROM Channels
LEFT JOIN Posts
	ON Posts.ChannelId = Channels.Id
	AND Posts.DeleteAt = 0
	AND Posts.CreateAt >= :window_start
	AND Posts.CreateAt < :window_end
	AND Posts.Type = ''
	AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
	AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
	AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
	AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
LEFT JOIN (
	SELECT ChannelMembers.ChannelId, count(*) AS MemberCount
	FROM ChannelMembers
	JOIN Channels AS MemberChannels
		ON MemberChannels.Id = ChannelMembers.ChannelId
		AND MemberChannels.TeamId = :'team_id'
	GROUP BY ChannelMembers.ChannelId
) AS Members ON Members.ChannelId = Channels.Id
LEFT JOIN (
	SELECT Posts.ChannelId, max(Posts.CreateAt) AS LastRealPostAt
	FROM Posts
	JOIN Channels AS PostChannels
		ON PostChannels.Id = Posts.ChannelId
		AND PostChannels.TeamId = :'team_id'
	WHERE Posts.DeleteAt = 0
		AND Posts.CreateAt >= :lookback_start
		AND Posts.CreateAt < :window_end
		AND Posts.Type = ''
		AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
		AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
		AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
		AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
	GROUP BY Posts.ChannelId
) AS LastReal ON LastReal.ChannelId = Channels.Id
WHERE Channels.TeamId = :'team_id'
	AND Channels.DeleteAt = 0
	AND (Channels.Type = 'O' OR Channels.Type = 'P')
GROUP BY
	Channels.Id, Channels.Type, Channels.DisplayName, Channels.Name,
	Channels.Purpose, Channels.Header, Channels.CreateAt,
	LastReal.LastRealPostAt, Members.MemberCount
ORDER BY MessageCount DESC, Channels.Name ASC;
\o

\echo ''
\echo '=== TIMED: current (MATERIALIZED CTE, shipped) ==='
\o /dev/null
WITH activity AS MATERIALIZED (
	SELECT
		Posts.ChannelId,
		count(*) FILTER (WHERE Posts.CreateAt >= :window_start) AS MessageCount,
		count(DISTINCT Posts.UserId) FILTER (WHERE Posts.CreateAt >= :window_start) AS ActivePosters,
		COALESCE(max(Posts.CreateAt) FILTER (WHERE Posts.CreateAt >= :window_start), 0) AS LastPostInWindow,
		max(Posts.CreateAt) AS LastRealPostAt
	FROM Posts
	WHERE Posts.DeleteAt = 0
		AND Posts.CreateAt >= :lookback_start
		AND Posts.CreateAt < :window_end
		AND Posts.Type = ''
		AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
		AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
		AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
		AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
	GROUP BY Posts.ChannelId
),
members AS MATERIALIZED (
	SELECT ChannelMembers.ChannelId, count(*) AS MemberCount
	FROM ChannelMembers
	JOIN Channels AS MemberChannels
		ON MemberChannels.Id = ChannelMembers.ChannelId
		AND MemberChannels.TeamId = :'team_id'
	GROUP BY ChannelMembers.ChannelId
)
SELECT
	Channels.Id,
	Channels.Type,
	Channels.DisplayName,
	Channels.Name,
	Channels.Purpose,
	Channels.Header,
	Channels.CreateAt,
	COALESCE(activity.LastRealPostAt, 0) AS LastPostAt,
	COALESCE(activity.LastPostInWindow, 0) AS LastPostInWindow,
	COALESCE(activity.MessageCount, 0) AS MessageCount,
	COALESCE(activity.ActivePosters, 0) AS ActivePosters,
	COALESCE(members.MemberCount, 0) AS MemberCount
FROM Channels
LEFT JOIN activity ON activity.ChannelId = Channels.Id
LEFT JOIN members ON members.ChannelId = Channels.Id
WHERE Channels.TeamId = :'team_id'
	AND Channels.DeleteAt = 0
	AND (Channels.Type = 'O' OR Channels.Type = 'P')
ORDER BY MessageCount DESC, Channels.Name ASC;
\o

-- ---------------------------------------------------------------------------
-- Plans. The "Time:" numbers above are the honest ones — EXPLAIN ANALYZE
-- instrumentation inflates a plan doing many loops over many rows, and the
-- legacy shape does exactly that.
--
-- What to look for in the legacy plan: a Nested Loop whose estimate is
-- rows=1 against an actual in the hundreds of thousands, feeding a
-- Materialize with loops= roughly your channel count. That is the whole bug.
-- ---------------------------------------------------------------------------

\echo ''
\echo '=== PLAN: legacy ==='
EXPLAIN (ANALYZE, BUFFERS)
SELECT
	Channels.Id,
	COALESCE(LastReal.LastRealPostAt, 0) AS LastPostAt,
	COALESCE(max(Posts.CreateAt), 0) AS LastPostInWindow,
	count(Posts.Id) AS MessageCount,
	count(DISTINCT Posts.UserId) AS ActivePosters,
	COALESCE(Members.MemberCount, 0) AS MemberCount
FROM Channels
LEFT JOIN Posts
	ON Posts.ChannelId = Channels.Id
	AND Posts.DeleteAt = 0
	AND Posts.CreateAt >= :window_start
	AND Posts.CreateAt < :window_end
	AND Posts.Type = ''
	AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
	AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
	AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
	AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
LEFT JOIN (
	SELECT ChannelMembers.ChannelId, count(*) AS MemberCount
	FROM ChannelMembers
	JOIN Channels AS MemberChannels
		ON MemberChannels.Id = ChannelMembers.ChannelId
		AND MemberChannels.TeamId = :'team_id'
	GROUP BY ChannelMembers.ChannelId
) AS Members ON Members.ChannelId = Channels.Id
LEFT JOIN (
	SELECT Posts.ChannelId, max(Posts.CreateAt) AS LastRealPostAt
	FROM Posts
	JOIN Channels AS PostChannels
		ON PostChannels.Id = Posts.ChannelId
		AND PostChannels.TeamId = :'team_id'
	WHERE Posts.DeleteAt = 0
		AND Posts.CreateAt >= :lookback_start
		AND Posts.CreateAt < :window_end
		AND Posts.Type = ''
		AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
		AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
		AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
		AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
	GROUP BY Posts.ChannelId
) AS LastReal ON LastReal.ChannelId = Channels.Id
WHERE Channels.TeamId = :'team_id'
	AND Channels.DeleteAt = 0
	AND (Channels.Type = 'O' OR Channels.Type = 'P')
GROUP BY Channels.Id, LastReal.LastRealPostAt, Members.MemberCount
ORDER BY MessageCount DESC;

\echo ''
\echo '=== PLAN: current ==='
EXPLAIN (ANALYZE, BUFFERS)
WITH activity AS MATERIALIZED (
	SELECT
		Posts.ChannelId,
		count(*) FILTER (WHERE Posts.CreateAt >= :window_start) AS MessageCount,
		count(DISTINCT Posts.UserId) FILTER (WHERE Posts.CreateAt >= :window_start) AS ActivePosters,
		COALESCE(max(Posts.CreateAt) FILTER (WHERE Posts.CreateAt >= :window_start), 0) AS LastPostInWindow,
		max(Posts.CreateAt) AS LastRealPostAt
	FROM Posts
	WHERE Posts.DeleteAt = 0
		AND Posts.CreateAt >= :lookback_start
		AND Posts.CreateAt < :window_end
		AND Posts.Type = ''
		AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
		AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
		AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
		AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
	GROUP BY Posts.ChannelId
),
members AS MATERIALIZED (
	SELECT ChannelMembers.ChannelId, count(*) AS MemberCount
	FROM ChannelMembers
	JOIN Channels AS MemberChannels
		ON MemberChannels.Id = ChannelMembers.ChannelId
		AND MemberChannels.TeamId = :'team_id'
	GROUP BY ChannelMembers.ChannelId
)
SELECT
	Channels.Id,
	COALESCE(activity.LastRealPostAt, 0) AS LastPostAt,
	COALESCE(activity.LastPostInWindow, 0) AS LastPostInWindow,
	COALESCE(activity.MessageCount, 0) AS MessageCount,
	COALESCE(activity.ActivePosters, 0) AS ActivePosters,
	COALESCE(members.MemberCount, 0) AS MemberCount
FROM Channels
LEFT JOIN activity ON activity.ChannelId = Channels.Id
LEFT JOIN members ON members.ChannelId = Channels.Id
WHERE Channels.TeamId = :'team_id'
	AND Channels.DeleteAt = 0
	AND (Channels.Type = 'O' OR Channels.Type = 'P')
ORDER BY MessageCount DESC;

\echo ''
\echo 'Done. Report the two "Time:" values, the ratio between them, and the'
\echo 'Dataset block above. A ratio without its dataset is not a result.'
