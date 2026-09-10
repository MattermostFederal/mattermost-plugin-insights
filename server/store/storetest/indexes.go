package storetest

import (
	"database/sql"
	"testing"
)

// productionIndexes mirrors the indexes real Mattermost has on the tables the
// channel-activity query touches, recovered from
// ../mattermost/server/channels/db/migrations/postgres/.
//
// schema.sql deliberately carries no indexes beyond primary keys, which is
// fine for correctness tests on six-row fixtures. It is not fine for a
// benchmark: without these, both query shapes sequential-scan everything and
// the comparison measures a database nobody runs.
//
// Only the indexes relevant to this query are listed. Notably absent is
// anything on Posts.Props — production has no expression index on the JSONB
// integration keys either, which is precisely why the planner misestimates
// those four predicates (INSIGHTS_REFERENCE.md §3.5).
var productionIndexes = []string{
	// posts: 000055_upgrade_posts_v6.0, 000041_create_posts
	`CREATE INDEX IF NOT EXISTS idx_posts_create_at ON posts(createat)`,
	`CREATE INDEX IF NOT EXISTS idx_posts_delete_at ON posts(deleteat)`,
	`CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(userid)`,
	`CREATE INDEX IF NOT EXISTS idx_posts_channel_id_delete_at_create_at ON posts(channelid, deleteat, createat)`,
	`CREATE INDEX IF NOT EXISTS idx_posts_channel_id_update_at ON posts(channelid, updateat)`,

	// channels: 000056_upgrade_channels_v6.0
	`CREATE INDEX IF NOT EXISTS idx_channels_team_id_display_name ON channels(teamid, displayname)`,
	`CREATE INDEX IF NOT EXISTS idx_channels_team_id_type ON channels(teamid, type)`,

	// channelmembers: 000058_upgrade_channelmembers_v6.0. Production indexes
	// (userid, channelid, lastviewedat); schema.sql has no lastviewedat
	// column, so the leading two columns are used.
	`CREATE INDEX IF NOT EXISTS idx_channelmembers_user_id_channel_id ON channelmembers(userid, channelid)`,
}

// ApplyProductionIndexes creates the production indexes on the shared test
// database. Intended for benchmarks; correctness tests do not need it and do
// not call it.
//
// Indexes outlive the TRUNCATE that NewDB does between tests, so this is
// idempotent and safe to call more than once per session.
func ApplyProductionIndexes(tb testing.TB, db *sql.DB) {
	tb.Helper()
	for _, stmt := range productionIndexes {
		if _, err := db.Exec(stmt); err != nil {
			tb.Fatalf("storetest: create index (%s): %v", stmt, err)
		}
	}
}
