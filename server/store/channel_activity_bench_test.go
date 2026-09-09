package store

// Benchmark comparing the two shapes of the channel-activity query on one
// synthetic dataset: the pre-d4d8604 row-by-row LEFT JOIN against the shipped
// MATERIALIZED CTE.
//
// This exists because the figures quoted in the d4d8604 commit message and in
// channel_activity.go's header comment were measured by hand and never
// recorded — no dataset, no plans, no methodology. Anything we say publicly
// about the speedup should come from a run of this instead.
//
// Report the RATIO, not the absolute seconds. A postgres:14-alpine container
// on a laptop differs from a production server on shared_buffers, work_mem
// and parallel workers, so absolute timings here do not transfer. The ratio
// and the plan shape do.
//
// Usage:
//
//	# one run of each shape, with plans
//	INSIGHTS_BENCH_EXPLAIN=1 go test ./server/store -run '^$' \
//	    -bench BenchmarkChannelActivityShapes -benchtime=1x -v
//
//	# five runs each, for benchstat
//	go test ./server/store -run '^$' -bench BenchmarkChannelActivityShapes \
//	    -benchtime=1x -count=5 -timeout=60m | tee new.txt
//
// Dataset size is controlled by the INSIGHTS_BENCH_* environment variables
// documented on benchConfig. Defaults approximate the team the original
// measurement claimed: ~490 channels on the target team, ~1M posts total.
//
// At the default size a -count=3 run takes a couple of minutes: ~10s to seed
// a million posts, then three runs of each shape, where the legacy one is
// expected to be slow — that is the point.
//
// A synthetic dataset only approximates a server. scripts/channel_activity_compare.sql
// runs the same two shapes against a real database and should be preferred
// whenever one is available.

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// legacyChannelActivityForTeamSQL is the query as it stood immediately before
// d4d8604, recovered verbatim with:
//
//	git show d4d8604^:server/store/channel_activity.go
//
// Kept byte-identical so the comparison is honest. Parameter order is
// (teamID, windowStart, windowEnd, lookbackStart) — note this differs from
// the current query, which takes (teamID, windowStart, lookbackStart,
// windowEnd).
const legacyChannelActivityForTeamSQL = `
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
	AND Posts.CreateAt >= $2
	AND Posts.CreateAt < $3
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
		AND MemberChannels.TeamId = $1
	GROUP BY ChannelMembers.ChannelId
) AS Members ON Members.ChannelId = Channels.Id
LEFT JOIN (
	SELECT Posts.ChannelId, max(Posts.CreateAt) AS LastRealPostAt
	FROM Posts
	JOIN Channels AS PostChannels
		ON PostChannels.Id = Posts.ChannelId
		AND PostChannels.TeamId = $1
	WHERE Posts.DeleteAt = 0
		AND Posts.CreateAt >= $4
		AND Posts.CreateAt < $3
		AND Posts.Type = ''
		AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
		AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
		AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
		AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
	GROUP BY Posts.ChannelId
) AS LastReal ON LastReal.ChannelId = Channels.Id
WHERE Channels.TeamId = $1
	AND Channels.DeleteAt = 0
	AND (Channels.Type = 'O' OR Channels.Type = 'P')
GROUP BY
	Channels.Id, Channels.Type, Channels.DisplayName, Channels.Name,
	Channels.Purpose, Channels.Header, Channels.CreateAt,
	LastReal.LastRealPostAt, Members.MemberCount
ORDER BY MessageCount DESC, Channels.Name ASC
`

// benchConfig is the dataset shape. Every field is overridable so the same
// benchmark can be pointed at a deployment's real numbers rather than our
// guesses — which matters, because the speedup is a function of the dataset
// and quoting one ratio as universal would be wrong.
type benchConfig struct {
	// TeamChannels is the channel count on the team being queried.
	// INSIGHTS_BENCH_CHANNELS.
	TeamChannels int
	// OtherTeams is how many additional teams exist. The shipped CTE
	// aggregates every channel on the server and discards the rest, so a
	// single-team dataset understates its cost and inflates the speedup.
	// INSIGHTS_BENCH_OTHER_TEAMS.
	OtherTeams int
	// TotalPosts is the post count across ALL channels on ALL teams, not
	// just the target team's share. INSIGHTS_BENCH_POSTS.
	TotalPosts int
	// IntegrationPct is the percentage of posts carrying a truthy
	// from_bot / from_webhook / from_oauth_app / from_plugin prop. These
	// are the predicates the planner cannot estimate through, so a dataset
	// with none of them will not reproduce the misestimate at all.
	// INSIGHTS_BENCH_INTEGRATION_PCT.
	IntegrationPct int
	// SystemPct is the percentage of posts with a non-empty Type (joins,
	// leaves, header changes). INSIGHTS_BENCH_SYSTEM_PCT.
	SystemPct int
	// DeletedPct is the percentage of soft-deleted posts.
	// INSIGHTS_BENCH_DELETED_PCT.
	DeletedPct int
	// MembersPerChannel is the average ChannelMembers rows per channel.
	// INSIGHTS_BENCH_MEMBERS.
	MembersPerChannel int
	// Users is the size of the user pool posts are attributed to.
	// INSIGHTS_BENCH_USERS.
	Users int
	// DMChannels is the number of direct- and group-message channels on the
	// server. They belong to no team and never appear in the result, but the
	// shipped CTE scans their posts along with everything else, so leaving
	// them out understates its cost. INSIGHTS_BENCH_DM_CHANNELS.
	DMChannels int
	// SkewPercent controls how unevenly posts are distributed across
	// channels, as the share of posts landing in the busiest 10% of
	// channels. Real servers are heavily skewed: a handful of busy channels
	// and a long tail of near-dead ones, which is the population the
	// governance table exists to surface. 10 means uniform.
	// INSIGHTS_BENCH_SKEW_PCT.
	SkewPercent int
	// RecencyExponent biases post timestamps toward the present. 1 spreads
	// them uniformly across the two-year lookback; higher values concentrate
	// them in recent months, as an active, growing server does. Expressed in
	// tenths to keep the env var an integer.
	// INSIGHTS_BENCH_RECENCY_TENTHS.
	RecencyTenths int
	// ReplyPercent is the share of posts that are thread replies (non-empty
	// RootId). Does not change the query's logic, but changes row width and
	// therefore scan cost. INSIGHTS_BENCH_REPLY_PCT.
	ReplyPercent int
	// AvgMessageChars is the mean length of the generated message text. Real
	// Posts rows are dominated by this column, and the query's cost is a
	// function of how many heap pages a scan has to read.
	// INSIGHTS_BENCH_AVG_MSG_CHARS.
	AvgMessageChars int
}

// defaultBenchConfig approximates the team described in channel_activity.go's
// header comment: ~490 channels, ~1M posts. The integration/system/deleted
// percentages are guesses — a real server's mix is the thing most worth
// overriding before quoting a number.
func defaultBenchConfig() benchConfig {
	return benchConfig{
		TeamChannels:      envInt("INSIGHTS_BENCH_CHANNELS", 490),
		OtherTeams:        envInt("INSIGHTS_BENCH_OTHER_TEAMS", 4),
		TotalPosts:        envInt("INSIGHTS_BENCH_POSTS", 1_000_000),
		IntegrationPct:    envInt("INSIGHTS_BENCH_INTEGRATION_PCT", 15),
		SystemPct:         envInt("INSIGHTS_BENCH_SYSTEM_PCT", 8),
		DeletedPct:        envInt("INSIGHTS_BENCH_DELETED_PCT", 2),
		MembersPerChannel: envInt("INSIGHTS_BENCH_MEMBERS", 25),
		Users:             envInt("INSIGHTS_BENCH_USERS", 2000),
		DMChannels:        envInt("INSIGHTS_BENCH_DM_CHANNELS", 5000),
		SkewPercent:       envInt("INSIGHTS_BENCH_SKEW_PCT", 60),
		RecencyTenths:     envInt("INSIGHTS_BENCH_RECENCY_TENTHS", 25),
		ReplyPercent:      envInt("INSIGHTS_BENCH_REPLY_PCT", 35),
		AvgMessageChars:   envInt("INSIGHTS_BENCH_AVG_MSG_CHARS", 120),
	}
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("%s: not an integer: %q", key, v))
	}
	return n
}

const benchTeamID = "bteam0aaaaaaaaaaaaaaaaaaaa"

// benchID builds a deterministic 26-character id, matching Mattermost's id
// width so index and row sizes are representative.
//
// The counter is left-padded to a fixed width rather than appended and the
// remainder filled: padding on the right with zeros makes benchID("u", 1) and
// benchID("u", 10) the same string, which surfaces as a primary-key
// violation partway through seeding.
func benchID(prefix string, n int) string {
	width := 26 - len(prefix)
	if width < 1 {
		panic("benchID: prefix too long: " + prefix)
	}
	s := fmt.Sprintf("%s%0*d", prefix, width, n)
	if len(s) > 26 {
		panic(fmt.Sprintf("benchID: %q overflows 26 characters", s))
	}
	return s
}

// BenchmarkChannelActivityShapes seeds one dataset and runs both query shapes
// against it. Both sub-benchmarks hit the same rows in the same database, so
// the ratio between them is the only number worth reporting.
func BenchmarkChannelActivityShapes(b *testing.B) {
	db := storetest.NewDB(b)
	cfg := defaultBenchConfig()
	window := benchWindow()

	seedBenchDataset(b, db, cfg, window)

	start, end := window.StartMillis(), window.EndMillis()
	lookback := start - lastPostLookbackMillis

	// Same window, different parameter order between the two shapes.
	legacyArgs := []any{benchTeamID, start, end, lookback}
	cteArgs := []any{benchTeamID, start, lookback, end}

	// Correctness gate. A speedup between queries that disagree is not a
	// speedup, and the legacy shape is no longer covered by any test.
	legacyRows := countRows(b, db, legacyChannelActivityForTeamSQL, legacyArgs...)
	cteRows := countRows(b, db, channelActivityForTeamSQL, cteArgs...)
	if legacyRows != cteRows {
		b.Fatalf("shapes disagree: legacy returned %d rows, CTE returned %d", legacyRows, cteRows)
	}
	b.Logf("dataset: %d team channels, %d other teams, %d posts, %d%% integration, %d%% system, %d%% deleted; both shapes return %d rows",
		cfg.TeamChannels, cfg.OtherTeams, cfg.TotalPosts,
		cfg.IntegrationPct, cfg.SystemPct, cfg.DeletedPct, legacyRows)

	if os.Getenv("INSIGHTS_BENCH_EXPLAIN") != "" {
		explain(b, db, "legacy", legacyChannelActivityForTeamSQL, legacyArgs...)
		explain(b, db, "cte", channelActivityForTeamSQL, cteArgs...)
	}

	b.Run("legacy_leftjoin", func(b *testing.B) {
		runShape(b, db, legacyChannelActivityForTeamSQL, legacyArgs...)
	})
	b.Run("cte_materialized", func(b *testing.B) {
		runShape(b, db, channelActivityForTeamSQL, cteArgs...)
	})
}

func runShape(b *testing.B, db *sql.DB, query string, args ...any) {
	b.Helper()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if n := countRows(b, db, query, args...); n == 0 {
			b.Fatal("query returned no rows")
		}
	}
}

// countRows runs the query and drains it without scanning into structs. The
// two shapes return the same columns, and struct scanning would add identical
// overhead to both, so it is left out.
func countRows(tb testing.TB, db *sql.DB, query string, args ...any) int {
	tb.Helper()
	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		tb.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	n := 0
	for rows.Next() {
		n++
	}
	if err := rows.Err(); err != nil {
		tb.Fatalf("rows: %v", err)
	}
	return n
}

// explain prints the plan. BUFFERS is included because the shared hit/read
// split is what distinguishes a warm run from a cold one — without it a fast
// number could just mean the data was already in cache.
func explain(tb testing.TB, db *sql.DB, label, query string, args ...any) {
	tb.Helper()
	rows, err := db.QueryContext(context.Background(),
		"EXPLAIN (ANALYZE, BUFFERS, VERBOSE) "+query, args...)
	if err != nil {
		tb.Fatalf("explain %s: %v", label, err)
	}
	defer func() { _ = rows.Close() }()

	var sb strings.Builder
	sb.WriteString("EXPLAIN (ANALYZE, BUFFERS) — " + label + "\n")
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			tb.Fatalf("explain scan: %v", err)
		}
		sb.WriteString(line + "\n")
	}
	tb.Log(sb.String())
}

// benchWindow returns a fixed 28-day closed window. It is pinned rather than
// derived from time.Now so repeated runs measure the same rows.
func benchWindow() insights.Window {
	end := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	return insights.Window{Start: end.AddDate(0, 0, -28), End: end}
}

// seedBenchDataset builds the whole fixture, then ANALYZEs.
//
// The ANALYZE is not optional. The entire pathology being measured is a
// planner misestimate — one row estimated where there were 900,000 — and
// without table statistics Postgres plans from defaults and the legacy shape
// may look artificially fine.
func seedBenchDataset(b *testing.B, db *sql.DB, cfg benchConfig, w insights.Window) {
	b.Helper()
	storetest.ApplyProductionIndexes(b, db)

	seedStart := time.Now()
	rng := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic fixture, not security

	users := make([]string, cfg.Users)
	for i := range users {
		users[i] = benchID("buser", i)
	}
	copyIn(b, db, "users", []string{"id", "username", "deleteat"},
		func(add func(...any)) {
			for i, u := range users {
				add(u, fmt.Sprintf("benchuser%d", i), 0)
			}
		})

	// Channels: the target team, the other teams, then the server's DM and
	// group-message channels. DMs belong to no team and never appear in the
	// result, but the shipped CTE scans their posts along with every other
	// channel's, so omitting them would measure a server that does not exist.
	type ch struct {
		id       string
		teamID   string
		chanType string
	}
	var channels []ch
	teams := []string{benchTeamID}
	for t := 0; t < cfg.OtherTeams; t++ {
		teams = append(teams, benchID("bteamx", t+1))
	}
	for ti, teamID := range teams {
		for c := 0; c < cfg.TeamChannels; c++ {
			// ~20% private, matching a typical mixed team.
			chanType := "O"
			if c%5 == 0 {
				chanType = "P"
			}
			channels = append(channels, ch{
				id:       benchID(fmt.Sprintf("bch%d_", ti), c),
				teamID:   teamID,
				chanType: chanType,
			})
		}
	}
	teamChannelCount := len(channels)
	for d := 0; d < cfg.DMChannels; d++ {
		chanType := "D"
		if d%10 == 0 {
			chanType = "G"
		}
		channels = append(channels, ch{id: benchID("bdm", d), teamID: "", chanType: chanType})
	}

	copyIn(b, db, "channels",
		[]string{"id", "type", "teamid", "displayname", "name", "purpose", "header", "createat", "lastpostat", "deleteat"},
		func(add func(...any)) {
			for i, c := range channels {
				deleteAt := 0
				if i%97 == 0 {
					deleteAt = 1
				}
				purpose := "benchmark channel"
				if i%3 == 0 {
					purpose = "" // unlabelled, as the governance table expects to find
				}
				name := fmt.Sprintf("bench-channel-%d", i)
				add(c.id, c.chanType, c.teamID, name, name, purpose, "", 1, 0, deleteAt)
			}
		})

	copyIn(b, db, "channelmembers", []string{"channelid", "userid"},
		func(add func(...any)) {
			for i, c := range channels {
				// DMs and group messages have a couple of members, not the
				// team-channel average.
				count := cfg.MembersPerChannel
				if i >= teamChannelCount {
					count = 2
					if c.chanType == "G" {
						count = 5
					}
				}
				startIdx := rng.Intn(len(users))
				for m := 0; m < count; m++ {
					add(c.id, users[(startIdx+m)%len(users)])
				}
			}
		})

	lookbackStart := w.StartMillis() - lastPostLookbackMillis
	span := w.EndMillis() - lookbackStart
	integrationKeys := []string{"from_bot", "from_webhook", "from_oauth_app", "from_plugin"}
	corpus := benchCorpus()
	recency := float64(cfg.RecencyTenths) / 10

	// The busiest 10% of channels take SkewPercent of all posts. Real servers
	// concentrate traffic in a handful of channels and leave a long tail of
	// near-dead ones — which is exactly the population the governance table
	// exists to surface, so a uniform spread would both flatter the query and
	// misrepresent the feature.
	hotCount := len(channels) / 10
	if hotCount < 1 {
		hotCount = 1
	}

	copyIn(b, db, "posts",
		[]string{"id", "userid", "channelid", "rootid", "createat", "updateat", "deleteat", "type", "props", "message", "hashtags"},
		func(add func(...any)) {
			for i := 0; i < cfg.TotalPosts; i++ {
				var c ch
				if rng.Intn(100) < cfg.SkewPercent {
					c = channels[rng.Intn(hotCount)]
				} else {
					c = channels[hotCount+rng.Intn(len(channels)-hotCount)]
				}

				// Timestamps bias toward the present: u^recency pushes the
				// sampled age toward zero, so most posts are recent and the
				// two-year lookback thins out, as on a growing server.
				age := int64(float64(span) * math.Pow(rng.Float64(), recency))
				createAt := w.EndMillis() - age
				if createAt < lookbackStart {
					createAt = lookbackStart
				}

				postType := ""
				if rng.Intn(100) < cfg.SystemPct {
					postType = "system_join_channel"
				}
				deleteAt := int64(0)
				if rng.Intn(100) < cfg.DeletedPct {
					deleteAt = createAt + 1000
				}
				rootID := ""
				if rng.Intn(100) < cfg.ReplyPercent {
					rootID = benchID("bpost", rng.Intn(cfg.TotalPosts))
				}

				// Props distribution is the load-bearing part of this
				// fixture. A mix of NULL, empty-object and truthy
				// integration flags is what makes the four `->>`
				// predicates unestimable, which is what produced the bad
				// plan in the first place.
				var props any
				switch {
				case rng.Intn(100) < cfg.IntegrationPct:
					k := integrationKeys[rng.Intn(len(integrationKeys))]
					props = fmt.Sprintf(`{%q: "true"}`, k)
				case rng.Intn(100) < 25:
					props = `{"disable_group_highlight": "false"}`
				default:
					props = nil
				}

				add(benchID("bpost", i), users[rng.Intn(len(users))], c.id, rootID,
					createAt, createAt, deleteAt, postType, props,
					benchMessage(rng, corpus, cfg.AvgMessageChars), "")
			}
		})

	if _, err := db.Exec(`ANALYZE posts, channels, channelmembers, users`); err != nil {
		b.Fatalf("analyze: %v", err)
	}

	var heapMB float64
	if err := db.QueryRow(`SELECT pg_total_relation_size('posts') / 1024.0 / 1024.0`).Scan(&heapMB); err != nil {
		b.Fatalf("posts size: %v", err)
	}
	b.Logf("seeded %d posts across %d channels (%d team, %d dm) — posts table %.0f MB — in %s",
		cfg.TotalPosts, len(channels), teamChannelCount, cfg.DMChannels, heapMB,
		time.Since(seedStart).Round(time.Second))
}

// benchCorpus returns filler prose to slice message bodies out of. Generating
// text word-by-word for a million rows is slow enough to matter; slicing a
// prebuilt string is not.
func benchCorpus() string {
	words := []string{
		"the", "release", "is", "blocked", "on", "review", "can", "someone", "take", "a",
		"look", "at", "this", "thread", "before", "standup", "tomorrow", "we", "should",
		"probably", "roll", "back", "and", "reland", "once", "tests", "are", "green",
		"deploy", "finished", "no", "errors", "in", "logs", "ping", "me", "if", "you",
		"see", "anything", "odd", "thanks", "for", "the", "quick", "turnaround", "here",
	}
	var sb strings.Builder
	for sb.Len() < 8192 {
		sb.WriteString(words[sb.Len()%len(words)])
		sb.WriteByte(' ')
	}
	return sb.String()
}

// benchMessage slices a message of roughly avg characters. Lengths vary
// because a table of uniform-width rows compresses and pages differently than
// a real one.
func benchMessage(rng *rand.Rand, corpus string, avg int) string {
	if avg <= 0 {
		return ""
	}
	n := 1 + rng.Intn(2*avg)
	if n > len(corpus)/2 {
		n = len(corpus) / 2
	}
	off := rng.Intn(len(corpus) - n)
	return corpus[off : off+n]
}

// copyIn bulk-loads a table via the COPY protocol. Row-at-a-time INSERTs of a
// million posts take long enough to discourage anyone from running this.
func copyIn(b *testing.B, db *sql.DB, table string, columns []string, rows func(add func(...any))) {
	b.Helper()

	tx, err := db.Begin()
	if err != nil {
		b.Fatalf("copyIn %s: begin: %v", table, err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(pq.CopyIn(table, columns...))
	if err != nil {
		b.Fatalf("copyIn %s: prepare: %v", table, err)
	}

	var addErr error
	rows(func(vals ...any) {
		if addErr != nil {
			return
		}
		if _, err := stmt.Exec(vals...); err != nil {
			addErr = err
		}
	})
	if addErr != nil {
		b.Fatalf("copyIn %s: buffer row: %v", table, addErr)
	}
	if _, err := stmt.Exec(); err != nil {
		b.Fatalf("copyIn %s: flush: %v", table, err)
	}
	if err := stmt.Close(); err != nil {
		b.Fatalf("copyIn %s: close: %v", table, err)
	}
	if err := tx.Commit(); err != nil {
		b.Fatalf("copyIn %s: commit: %v", table, err)
	}
}
