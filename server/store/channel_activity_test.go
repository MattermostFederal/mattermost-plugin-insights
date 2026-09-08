package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// Fixture shape — one team, six channels covering every case the governance
// table has to get right:
//
//	activeCh   public,  purpose set,   3 posts by 2 distinct users, 2 members
//	privateCh  private, purpose set,   1 post  by 1 user,           1 member
//	abandonCh  public,  purpose set,   0 posts,                     3 members
//	unlabelled public,  no purpose,    1 post,                      1 member
//	staleCh    public,  purpose set,   1 post BEFORE the window,    1 member
//	botCh      public,  purpose set,   1 post from a bot,           1 member
//
// Plus a deleted channel and a channel on another team, both of which must
// not appear at all.

const (
	activeChID     = "cact0aaaaaaaaaaaaaaaaaaaaa"
	privateChID    = "cact1aaaaaaaaaaaaaaaaaaaaa"
	abandonChID    = "cact2aaaaaaaaaaaaaaaaaaaaa"
	unlabelledChID = "cact3aaaaaaaaaaaaaaaaaaaaa"
	staleChID      = "cact4aaaaaaaaaaaaaaaaaaaaa"
	botChID        = "cact5aaaaaaaaaaaaaaaaaaaaa"
	deletedChID    = "cact6aaaaaaaaaaaaaaaaaaaaa"
	otherTeamChID  = "cact7aaaaaaaaaaaaaaaaaaaaa"

	otherTeamID = "team9aaaaaaaaaaaaaaaaaaaaa"
)

func seedActivityChannel(t *testing.T, db *sql.DB, id, chanType, teamID, name, purpose string, createAt, lastPostAt, deleteAt int64) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO channels (id, type, teamid, displayname, name, purpose, header, createat, lastpostat, deleteat)
		 VALUES ($1, $2, $3, $4, $4, $5, '', $6, $7, $8)`,
		id, chanType, teamID, name, purpose, createAt, lastPostAt, deleteAt)
	if chanType == "O" {
		mustExec(t, db,
			`INSERT INTO publicchannels (id, teamid, displayname, name, deleteat) VALUES ($1, $2, $3, $3, $4)`,
			id, teamID, name, deleteAt)
	}
}

func seedMembers(t *testing.T, db *sql.DB, channelID string, userIDs ...string) {
	t.Helper()
	for _, u := range userIDs {
		mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, channelID, u)
	}
}

// seedChannelActivityFixture returns the window cutoff the tests should pass
// as `since`.
func seedChannelActivityFixture(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	now := nowMillis()
	since := now - 1000
	beforeWindow := since - 10_000
	postIDs := postIDGen('a')

	user3 := "user3aaaaaaaaaaaaaaaaaaaaa"

	seedActivityChannel(t, db, activeChID, "O", testTeamID, "active", "team standup", 100, now, 0)
	seedMembers(t, db, activeChID, testUser1ID, testUser2ID)
	seedPostBy(t, db, postIDs(), testUser1ID, activeChID, now)
	seedPostBy(t, db, postIDs(), testUser1ID, activeChID, now)
	seedPostBy(t, db, postIDs(), testUser2ID, activeChID, now)

	seedActivityChannel(t, db, privateChID, "P", testTeamID, "private", "leads only", 200, now, 0)
	seedMembers(t, db, privateChID, testUser1ID)
	seedPostBy(t, db, postIDs(), testUser1ID, privateChID, now)

	seedActivityChannel(t, db, abandonChID, "O", testTeamID, "abandoned", "old project", 300, 0, 0)
	seedMembers(t, db, abandonChID, testUser1ID, testUser2ID, user3)

	seedActivityChannel(t, db, unlabelledChID, "O", testTeamID, "unlabelled", "", 400, now, 0)
	seedMembers(t, db, unlabelledChID, testUser1ID)
	seedPostBy(t, db, postIDs(), testUser1ID, unlabelledChID, now)

	seedActivityChannel(t, db, staleChID, "O", testTeamID, "stale", "dormant", 500, beforeWindow, 0)
	seedMembers(t, db, staleChID, testUser1ID)
	seedPostBy(t, db, postIDs(), testUser1ID, staleChID, beforeWindow)

	seedActivityChannel(t, db, botChID, "O", testTeamID, "botfeed", "alerts", 600, now, 0)
	seedMembers(t, db, botChID, testUser1ID)
	botPost := postIDs()
	mustExec(t, db,
		`INSERT INTO posts (id, userid, channelid, createat, deleteat, type, props)
		 VALUES ($1, $2, $3, $4, 0, '', '{"from_bot":"true"}'::jsonb)`,
		botPost, testUser1ID, botChID, now)

	seedActivityChannel(t, db, deletedChID, "O", testTeamID, "deleted", "gone", 700, now, now)
	seedActivityChannel(t, db, otherTeamChID, "O", otherTeamID, "elsewhere", "other", 800, now, 0)

	return since
}

func activityByID(t *testing.T, db *sql.DB, since int64) map[string]*struct {
	Purpose                            string
	MessageCount, Posters, MemberCount int64
	LastPostAt                         int64
	Type                               string
} {
	t.Helper()
	s := NewFromDB(db)
	rows, err := s.ChannelActivityForTeam(context.Background(), testTeamID, since)
	if err != nil {
		t.Fatalf("ChannelActivityForTeam: %v", err)
	}
	out := make(map[string]*struct {
		Purpose                            string
		MessageCount, Posters, MemberCount int64
		LastPostAt                         int64
		Type                               string
	}, len(rows))
	for _, r := range rows {
		out[r.ID] = &struct {
			Purpose                            string
			MessageCount, Posters, MemberCount int64
			LastPostAt                         int64
			Type                               string
		}{r.Purpose, r.MessageCount, r.ActivePosters, r.MemberCount, r.LastPostAt, string(r.Type)}
	}
	return out
}

func TestStore_ChannelActivityForTeam_countsAndMetadata(t *testing.T) {
	db := storetest.NewDB(t)
	since := seedChannelActivityFixture(t, db)
	got := activityByID(t, db, since)

	cases := []struct {
		name                               string
		id                                 string
		wantPurpose                        string
		wantMessages, wantPosters, wantMem int64
	}{
		{"active channel counts distinct posters", activeChID, "team standup", 3, 2, 2},
		{"private channel is included unfiltered", privateChID, "leads only", 1, 1, 1},
		{"abandoned channel appears with zero posts", abandonChID, "old project", 0, 0, 3},
		{"unlabelled channel has empty purpose", unlabelledChID, "", 1, 1, 1},
		{"posts before the window do not count", staleChID, "dormant", 0, 0, 1},
		{"bot posts are excluded", botChID, "alerts", 0, 0, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row, ok := got[tc.id]
			if !ok {
				t.Fatalf("channel %s missing from results", tc.id)
			}
			if row.Purpose != tc.wantPurpose {
				t.Errorf("Purpose = %q; want %q", row.Purpose, tc.wantPurpose)
			}
			if row.MessageCount != tc.wantMessages {
				t.Errorf("MessageCount = %d; want %d", row.MessageCount, tc.wantMessages)
			}
			if row.Posters != tc.wantPosters {
				t.Errorf("ActivePosters = %d; want %d", row.Posters, tc.wantPosters)
			}
			if row.MemberCount != tc.wantMem {
				t.Errorf("MemberCount = %d; want %d", row.MemberCount, tc.wantMem)
			}
		})
	}
}

func TestStore_ChannelActivityForTeam_excludesDeletedAndOtherTeams(t *testing.T) {
	db := storetest.NewDB(t)
	since := seedChannelActivityFixture(t, db)
	got := activityByID(t, db, since)

	if _, ok := got[deletedChID]; ok {
		t.Error("deleted channel should not appear")
	}
	if _, ok := got[otherTeamChID]; ok {
		t.Error("channel from another team should not appear")
	}
	if len(got) != 6 {
		t.Errorf("got %d channels; want 6", len(got))
	}
}

// LastPostAt is read straight off Channels rather than derived from Posts, so
// it reflects real activity even when that activity falls outside the window.
// The governance table uses it to answer "when was this last touched at all".
func TestStore_ChannelActivityForTeam_reportsLastPostAtOutsideWindow(t *testing.T) {
	db := storetest.NewDB(t)
	since := seedChannelActivityFixture(t, db)
	got := activityByID(t, db, since)

	stale, ok := got[staleChID]
	if !ok {
		t.Fatal("stale channel missing")
	}
	if stale.LastPostAt == 0 || stale.LastPostAt >= since {
		t.Errorf("LastPostAt = %d; want a nonzero value before since=%d", stale.LastPostAt, since)
	}
	if abandoned := got[abandonChID]; abandoned.LastPostAt != 0 {
		t.Errorf("never-posted channel LastPostAt = %d; want 0", abandoned.LastPostAt)
	}
}

func TestStore_ChannelActivityForTeam_returnsChannelType(t *testing.T) {
	db := storetest.NewDB(t)
	since := seedChannelActivityFixture(t, db)
	got := activityByID(t, db, since)

	if got[activeChID].Type != "O" {
		t.Errorf("public channel Type = %q; want O", got[activeChID].Type)
	}
	if got[privateChID].Type != "P" {
		t.Errorf("private channel Type = %q; want P", got[privateChID].Type)
	}
}
