package store

import (
	"context"
	"database/sql"
	"slices"
	"sort"
	"testing"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// Fixtures adapted from server/channels/api4/insights_test.go's
// TestGetTopInactiveChannelsForTeamSince in commit 26617fcbdc. The original
// helpers create a mix of public and private channels with varying activity.
//
// We hand-roll a simpler equivalent: 5 channels in a team, each with a known
// message count, and assert the ascending ordering (least active first).

func seedChannelWithCreateAt(t *testing.T, db *sql.DB, id, chanType, teamID, displayName string, createAt int64) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO channels (id, type, teamid, displayname, name, createat) VALUES ($1, $2, $3, $4, $4, $5)`,
		id, chanType, teamID, displayName, createAt)
	if chanType == "O" {
		mustExec(t, db,
			`INSERT INTO publicchannels (id, teamid, displayname, name) VALUES ($1, $2, $3, $3)`,
			id, teamID, displayName)
	}
}

type inactiveFixture struct {
	channels  []string // ids in expected ASC order (least active first)
	createdAt int64
}

func seedInactiveChannelFixture(t *testing.T, db *sql.DB) inactiveFixture {
	t.Helper()
	now := nowMillis()
	channelCreatedAt := now - 7*24*60*60*1000 // a week ago
	postIDs := postIDGen('q')

	type spec struct {
		id, kind, name string
		posts          int
	}
	specs := []spec{
		{"chQ0aaaaaaaaaaaaaaaaaaaaaa", "O", "alpha", 0}, // zero posts → least active
		{"chQ1aaaaaaaaaaaaaaaaaaaaaa", "O", "bravo", 1},
		{"chQ2aaaaaaaaaaaaaaaaaaaaaa", "P", "charlie", 2},
		{"chQ3aaaaaaaaaaaaaaaaaaaaaa", "O", "delta", 3},
		{"chQ4aaaaaaaaaaaaaaaaaaaaaa", "P", "echo", 5},
	}

	for _, sp := range specs {
		seedChannelWithCreateAt(t, db, sp.id, sp.kind, testTeamID, sp.name, channelCreatedAt)
		mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, sp.id, testUser1ID)
		for range sp.posts {
			seedPostBy(t, db, postIDs(), testUser1ID, sp.id, now)
		}
	}

	ids := make([]string, 0, len(specs))
	for _, sp := range specs {
		ids = append(ids, sp.id)
	}
	return inactiveFixture{channels: ids, createdAt: channelCreatedAt}
}

func TestStore_TopInactiveChannelsForTeamSince_orderedAscending(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	fix := seedInactiveChannelFixture(t, db)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopInactiveChannelsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 5)
	if err != nil {
		t.Fatalf("TopInactiveChannelsForTeamSince: %v", err)
	}
	if len(got.Items) != 5 {
		t.Fatalf("expected 5 items; got %d: %#v", len(got.Items), got.Items)
	}
	wantCounts := []int64{0, 1, 2, 3, 5}
	for i, item := range got.Items {
		if item.ID != fix.channels[i] {
			t.Errorf("rank %d: got id %s; want %s", i, item.ID, fix.channels[i])
		}
		if item.MessageCount != wantCounts[i] {
			t.Errorf("rank %d: count = %d; want %d (item=%#v)", i, item.MessageCount, wantCounts[i], item)
		}
	}
}

func TestStore_TopInactiveChannelsForTeamSince_excludesChannelsCreatedAfterWindow(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	// Old channel — qualifies.
	seedChannelWithCreateAt(t, db, "oldchanaaaaaaaaaaaaaaaaaaa", "O", testTeamID, "old", now-7*24*60*60*1000)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "oldchanaaaaaaaaaaaaaaaaaaa", testUser1ID)

	// Brand-new channel — should be excluded (createdAt >= since).
	seedChannelWithCreateAt(t, db, "newchanaaaaaaaaaaaaaaaaaaa", "O", testTeamID, "new", now)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "newchanaaaaaaaaaaaaaaaaaaa", testUser1ID)

	got, err := s.TopInactiveChannelsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopInactiveChannelsForTeamSince: %v", err)
	}
	for _, item := range got.Items {
		if item.ID == "newchanaaaaaaaaaaaaaaaaaaa" {
			t.Fatalf("brand-new channel leaked into inactive list: %#v", got.Items)
		}
	}
	found := false
	for _, item := range got.Items {
		if item.ID == "oldchanaaaaaaaaaaaaaaaaaaa" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected old channel to be present; got %#v", got.Items)
	}
}

func TestStore_TopInactiveChannelsForTeamSince_excludesPrivateChannelsUserIsNotMemberOf(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000
	old := now - 7*24*60*60*1000

	// Private channel that user1 is NOT a member of.
	seedChannelWithCreateAt(t, db, "secretaaaaaaaaaaaaaaaaaaaa", "P", testTeamID, "secret", old)

	got, err := s.TopInactiveChannelsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopInactiveChannelsForTeamSince: %v", err)
	}
	for _, item := range got.Items {
		if item.ID == "secretaaaaaaaaaaaaaaaaaaaa" {
			t.Fatalf("private channel user is not a member of leaked into result: %#v", got.Items)
		}
	}
}

func TestStore_TopInactiveChannelsForTeamSince_populatesParticipants(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000
	old := now - 7*24*60*60*1000

	seedChannelWithCreateAt(t, db, "chan0aaaaaaaaaaaaaaaaaaaaa", "O", testTeamID, "general", old)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "chan0aaaaaaaaaaaaaaaaaaaaa", testUser1ID)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "chan0aaaaaaaaaaaaaaaaaaaaa", testUser2ID)

	got, err := s.TopInactiveChannelsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopInactiveChannelsForTeamSince: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 channel; got %#v", got.Items)
	}
	parts := []string{}
	parts = append(parts, got.Items[0].Participants...)
	sort.Strings(parts)
	want := []string{testUser1ID, testUser2ID}
	sort.Strings(want)
	if !slices.Equal(parts, want) {
		t.Errorf("participants = %v; want %v", parts, want)
	}
}

func TestStore_TopInactiveChannelsForUserSince_excludesNonMember(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000
	old := now - 7*24*60*60*1000

	seedChannelWithCreateAt(t, db, "memchanaaaaaaaaaaaaaaaaaaa", "O", testTeamID, "member-of", old)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "memchanaaaaaaaaaaaaaaaaaaa", testUser1ID)

	seedChannelWithCreateAt(t, db, "outchanaaaaaaaaaaaaaaaaaaa", "O", testTeamID, "not-member", old)
	// no membership for user1

	got, err := s.TopInactiveChannelsForUserSince(context.Background(), testUser1ID, testTeamID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopInactiveChannelsForUserSince: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "memchanaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("got %#v; want only memchan", got.Items)
	}
}

func TestStore_TopInactiveChannelsForUserSince_excludesDMs(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000
	old := now - 7*24*60*60*1000

	seedChannelWithCreateAt(t, db, "memchanaaaaaaaaaaaaaaaaaaa", "O", testTeamID, "member-of", old)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "memchanaaaaaaaaaaaaaaaaaaa", testUser1ID)

	mustExec(t, db, `INSERT INTO channels (id, type, teamid, name, createat) VALUES ($1, 'D', '', $1, $2)`, "dmchanaaaaaaaaaaaaaaaaaaaa", old)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "dmchanaaaaaaaaaaaaaaaaaaaa", testUser1ID)

	got, err := s.TopInactiveChannelsForUserSince(context.Background(), testUser1ID, "", since, 0, 10)
	if err != nil {
		t.Fatalf("TopInactiveChannelsForUserSince: %v", err)
	}
	for _, item := range got.Items {
		if item.ID == "dmchanaaaaaaaaaaaaaaaaaaaa" {
			t.Fatalf("DM leaked into inactive channels: %#v", got.Items)
		}
	}
}
