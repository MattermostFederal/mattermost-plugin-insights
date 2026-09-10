package store

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// Fixtures adapted from server/channels/api4/insights_test.go's
// TestGetTopChannelsForTeamSince in commit 26617fcbdc:
//
//   6 channels in the team, all reachable to user1:
//     channel0 public      -> 6 user posts
//     channel1 public      -> 5
//     channel2 private (m) -> 4
//     channel3 public      -> 3
//     channel4 private (m) -> 2
//     channel5 private (m) -> 1
//
//   page 0 of 5 returns channel0..channel4 (descending by MessageCount).
//   page 1 returns channel5.

func seedChannel(t *testing.T, db *sql.DB, id, chanType, teamID, displayName string) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO channels (id, type, teamid, displayname, name) VALUES ($1, $2, $3, $4, $4)`,
		id, chanType, teamID, displayName)
	if chanType == "O" {
		mustExec(t, db,
			`INSERT INTO publicchannels (id, teamid, displayname, name) VALUES ($1, $2, $3, $3)`,
			id, teamID, displayName)
	}
}

func seedPostBy(t *testing.T, db *sql.DB, postID, userID, channelID string, createAt int64) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO posts (id, userid, channelid, createat, deleteat, type) VALUES ($1, $2, $3, $4, 0, '')`,
		postID, userID, channelID, createAt)
}

type channelFixture struct {
	channelIDs []string
}

func seedChannelFixture(t *testing.T, db *sql.DB) channelFixture {
	t.Helper()
	now := nowMillis()
	postIDs := postIDGen()

	type chanSpec struct {
		id, kind, name string
	}
	channels := []chanSpec{
		{"chan0aaaaaaaaaaaaaaaaaaaaa", "O", "alpha"},
		{"chan1aaaaaaaaaaaaaaaaaaaaa", "O", "bravo"},
		{"chan2aaaaaaaaaaaaaaaaaaaaa", "P", "charlie"},
		{"chan3aaaaaaaaaaaaaaaaaaaaa", "O", "delta"},
		{"chan4aaaaaaaaaaaaaaaaaaaaa", "P", "echo"},
		{"chan5aaaaaaaaaaaaaaaaaaaaa", "P", "foxtrot"},
	}

	postsByChannel := []int{6, 5, 4, 3, 2, 1}
	for i, ch := range channels {
		seedChannel(t, db, ch.id, ch.kind, testTeamID, ch.name)
		mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, ch.id, testUser1ID)
		for j := 0; j < postsByChannel[i]; j++ {
			seedPostBy(t, db, postIDs(), testUser1ID, ch.id, now)
		}
	}

	ids := make([]string, 0, len(channels))
	for _, c := range channels {
		ids = append(ids, c.id)
	}
	return channelFixture{channelIDs: ids}
}

func TestStore_TopChannelsForTeamSince_basicOrdering(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	fix := seedChannelFixture(t, db)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopChannelsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 5)
	if err != nil {
		t.Fatalf("TopChannelsForTeamSince: %v", err)
	}
	if len(got.Items) != 5 {
		t.Fatalf("expected 5 items; got %d: %#v", len(got.Items), got.Items)
	}
	wantIDs := fix.channelIDs[:5]
	wantCounts := []int64{6, 5, 4, 3, 2}
	for i, item := range got.Items {
		if item.ID != wantIDs[i] || item.MessageCount != wantCounts[i] {
			t.Fatalf("rank %d: got %s=%d; want %s=%d", i, item.ID, item.MessageCount, wantIDs[i], wantCounts[i])
		}
	}
	if !got.HasNext {
		t.Errorf("hasNext = false; want true (one more channel on page 1)")
	}
}

func TestStore_TopChannelsForTeamSince_pagination(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	fix := seedChannelFixture(t, db)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopChannelsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 1, 5)
	if err != nil {
		t.Fatalf("TopChannelsForTeamSince page=1: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != fix.channelIDs[5] {
		t.Fatalf("page 1 = %#v; want only %s", got.Items, fix.channelIDs[5])
	}
	if got.HasNext {
		t.Errorf("hasNext = true on last page; want false")
	}
}

func TestStore_TopChannelsForTeamSince_excludesPrivateChannelsUserIsNotMemberOf(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	seedChannelFixture(t, db)

	now := nowMillis()
	since := now - 24*60*60*1000
	postIDs := postIDGen('x')

	// Add a busy private channel that user1 is NOT a member of.
	seedChannel(t, db, "excluchanaaaaaaaaaaaaaaaaa", "P", testTeamID, "secret")
	for range 100 {
		seedPostBy(t, db, postIDs(), testUser2ID, "excluchanaaaaaaaaaaaaaaaaa", now)
	}

	got, err := s.TopChannelsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 5)
	if err != nil {
		t.Fatalf("TopChannelsForTeamSince: %v", err)
	}
	for _, item := range got.Items {
		if item.ID == "excluchanaaaaaaaaaaaaaaaaa" {
			t.Fatalf("private channel user is not a member of leaked into team channels: %#v", got.Items)
		}
	}
}

func TestStore_TopChannelsForTeamSince_excludesBotPosts(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	seedChannel(t, db, testPublicChanID, "O", testTeamID, "general")
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, testPublicChanID, testUser1ID)

	postIDs := postIDGen()
	mustExec(t, db,
		`INSERT INTO posts (id, userid, channelid, createat, deleteat, type, props) VALUES ($1, $2, $3, $4, 0, '', $5)`,
		postIDs(), testUser2ID, testPublicChanID, now, `{"from_bot":"true"}`)
	mustExec(t, db,
		`INSERT INTO posts (id, userid, channelid, createat, deleteat, type, props) VALUES ($1, $2, $3, $4, 0, '', $5)`,
		postIDs(), testUser2ID, testPublicChanID, now, `{"from_webhook":"true"}`)
	seedPostBy(t, db, postIDs(), testUser2ID, testPublicChanID, now)

	got, err := s.TopChannelsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 5)
	if err != nil {
		t.Fatalf("TopChannelsForTeamSince: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != testPublicChanID {
		t.Fatalf("got %#v", got.Items)
	}
	if got.Items[0].MessageCount != 1 {
		t.Errorf("expected 1 (only the human post); got %d", got.Items[0].MessageCount)
	}
}

func TestStore_TopChannelsForUserSince_onlyOwnPostsInMemberChannels(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000
	postIDs := postIDGen()

	// channel A: user1 member, posts by user1 + user2
	seedChannel(t, db, "chanaaaaaaaaaaaaaaaaaaaaaa", "O", testTeamID, "team-a")
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "chanaaaaaaaaaaaaaaaaaaaaaa", testUser1ID)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "chanaaaaaaaaaaaaaaaaaaaaaa", testUser2ID)
	for range 3 {
		seedPostBy(t, db, postIDs(), testUser1ID, "chanaaaaaaaaaaaaaaaaaaaaaa", now)
	}
	for range 5 {
		seedPostBy(t, db, postIDs(), testUser2ID, "chanaaaaaaaaaaaaaaaaaaaaaa", now)
	}

	// channel B: user1 member, posts by user1 only
	seedChannel(t, db, "chanbaaaaaaaaaaaaaaaaaaaaa", "O", testTeamID, "team-b")
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "chanbaaaaaaaaaaaaaaaaaaaaa", testUser1ID)
	for range 4 {
		seedPostBy(t, db, postIDs(), testUser1ID, "chanbaaaaaaaaaaaaaaaaaaaaa", now)
	}

	// channel C: user1 NOT a member, even though they "posted" (shouldn't happen in MM, but protect against it)
	seedChannel(t, db, "chancaaaaaaaaaaaaaaaaaaaaa", "O", testTeamID, "team-c")
	for range 10 {
		seedPostBy(t, db, postIDs(), testUser1ID, "chancaaaaaaaaaaaaaaaaaaaaa", now)
	}

	got, err := s.TopChannelsForUserSince(context.Background(), testUser1ID, "", since, now+1, 0, 5)
	if err != nil {
		t.Fatalf("TopChannelsForUserSince: %v", err)
	}
	counts := map[string]int64{}
	for _, item := range got.Items {
		counts[item.ID] = item.MessageCount
	}
	if counts["chanaaaaaaaaaaaaaaaaaaaaaa"] != 3 {
		t.Errorf("channel A count = %d; want 3 (only user1's posts)", counts["chanaaaaaaaaaaaaaaaaaaaaaa"])
	}
	if counts["chanbaaaaaaaaaaaaaaaaaaaaa"] != 4 {
		t.Errorf("channel B count = %d; want 4", counts["chanbaaaaaaaaaaaaaaaaaaaaa"])
	}
	if _, present := counts["chancaaaaaaaaaaaaaaaaaaaaa"]; present {
		t.Errorf("channel C should be excluded (non-member); got %d", counts["chancaaaaaaaaaaaaaaaaaaaaa"])
	}
}

func TestStore_TopChannelsForUserSince_teamFilter(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000
	postIDs := postIDGen()

	seedChannel(t, db, "chinaaaaaaaaaaaaaaaaaaaaaa", "O", testTeamID, "in-team")
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "chinaaaaaaaaaaaaaaaaaaaaaa", testUser1ID)
	seedChannel(t, db, "choutaaaaaaaaaaaaaaaaaaaaa", "O", testOtherTeamID, "out-of-team")
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "choutaaaaaaaaaaaaaaaaaaaaa", testUser1ID)

	for range 2 {
		seedPostBy(t, db, postIDs(), testUser1ID, "chinaaaaaaaaaaaaaaaaaaaaaa", now)
	}
	for range 3 {
		seedPostBy(t, db, postIDs(), testUser1ID, "choutaaaaaaaaaaaaaaaaaaaaa", now)
	}

	got, err := s.TopChannelsForUserSince(context.Background(), testUser1ID, testTeamID, since, now+1, 0, 10)
	if err != nil {
		t.Fatalf("TopChannelsForUserSince: %v", err)
	}
	for _, item := range got.Items {
		if item.ID == "choutaaaaaaaaaaaaaaaaaaaaa" {
			t.Errorf("channel from a different team leaked into result: %#v", got.Items)
		}
	}
	if len(got.Items) != 1 || got.Items[0].ID != "chinaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("got %#v; want exactly the in-team channel", got.Items)
	}
}

func TestStore_TopChannelsForUserSince_usesClosedWindow(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	start := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC).UnixMilli()
	end := time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC).UnixMilli()
	channelID := "windowaaaaaaaaaaaaaaaaaaaa"
	seedChannel(t, db, channelID, "O", testTeamID, "window")
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, channelID, testUser1ID)
	postIDs := postIDGen('w')
	for _, createAt := range []int64{start - 1, start, end - 1, end} {
		seedPostBy(t, db, postIDs(), testUser1ID, channelID, createAt)
	}

	got, err := s.TopChannelsForUserSince(context.Background(), testUser1ID, testTeamID, start, end, 0, 10)
	if err != nil {
		t.Fatalf("TopChannelsForUserSince: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].MessageCount != 2 {
		t.Fatalf("got %#v; want the posts at start and immediately before end", got.Items)
	}
}

// suppress fmt-unused warning until we add more Sprintf-using helpers.
var _ = fmt.Sprintf
