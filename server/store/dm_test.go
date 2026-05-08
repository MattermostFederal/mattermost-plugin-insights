package store

import (
	"context"
	"database/sql"
	"sort"
	"testing"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// dmName returns a DM channel name in the canonical Mattermost shape
// `<sortedUserIdA>__<sortedUserIdB>`.
func dmName(a, b string) string {
	if a < b {
		return a + "__" + b
	}
	return b + "__" + a
}

func seedDMChannel(t *testing.T, db *sql.DB, channelID, userA, userB string) {
	t.Helper()
	mustExec(t, db, `INSERT INTO channels (id, type, teamid, name) VALUES ($1, 'D', '', $2)`, channelID, dmName(userA, userB))
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, channelID, userA)
	if userA != userB {
		mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, channelID, userB)
	}
}

// seedDMPost inserts a post in a DM channel with both createat and updateat
// set to the given timestamp (DM query filters on updateat).
func seedDMPost(t *testing.T, db *sql.DB, postID, userID, channelID string, ts int64) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO posts (id, userid, channelid, createat, updateat, deleteat, type) VALUES ($1, $2, $3, $4, $4, 0, '')`,
		postID, userID, channelID, ts)
}

func TestStore_TopDMsForUserSince_basicCounts(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000
	postIDs := postIDGen('d')

	// 3 DM partners with varying message counts.
	partners := []struct {
		channelID, userID    string
		fromUser1, fromOther int
	}{
		{"dmaaaaaaaaaaaaaaaaaaaaaaaa", testUser2ID, 4, 6},                        // total 10
		{"dmbbbbbbbbbbbbbbbbbbbbbbbb", testUser3ID, 1, 1},                        // total 2
		{"dmccccccccccccccccccccccccc"[:26], "user4aaaaaaaaaaaaaaaaaaaaa", 3, 0}, // total 3
	}
	for _, p := range partners {
		seedDMChannel(t, db, p.channelID, testUser1ID, p.userID)
		for range p.fromUser1 {
			seedDMPost(t, db, postIDs(), testUser1ID, p.channelID, now)
		}
		for range p.fromOther {
			seedDMPost(t, db, postIDs(), p.userID, p.channelID, now)
		}
	}

	got, err := s.TopDMsForUserSince(context.Background(), testUser1ID, since, 0, 5)
	if err != nil {
		t.Fatalf("TopDMsForUserSince: %v", err)
	}
	if len(got.Items) != 3 {
		t.Fatalf("expected 3 DMs; got %d: %#v", len(got.Items), got.Items)
	}
	wantOrder := []string{"dmaaaaaaaaaaaaaaaaaaaaaaaa", "dmccccccccccccccccccccccccc"[:26], "dmbbbbbbbbbbbbbbbbbbbbbbbb"}
	for i, item := range got.Items {
		if item.ChannelID != wantOrder[i] {
			t.Errorf("rank %d: channel = %s; want %s (msg count %d)", i, item.ChannelID, wantOrder[i], item.MessageCount)
		}
	}
	// Top DM was 4+6=10 raw posts, but the query counts each post twice
	// (one row per channel-member pair), so the store returns 10 from the
	// SQL — the divide-by-2 happens in handler post-processing.
}

func TestStore_TopDMsForUserSince_excludesSelfDM(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000
	postIDs := postIDGen('s')

	// Self DM (name is "userId__userId").
	mustExec(t, db, `INSERT INTO channels (id, type, teamid, name) VALUES ($1, 'D', '', $2)`, "selfdmaaaaaaaaaaaaaaaaaaaa", testUser1ID+"__"+testUser1ID)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, "selfdmaaaaaaaaaaaaaaaaaaaa", testUser1ID)
	for range 5 {
		seedDMPost(t, db, postIDs(), testUser1ID, "selfdmaaaaaaaaaaaaaaaaaaaa", now)
	}

	// Regular DM that should appear.
	seedDMChannel(t, db, "regdmaaaaaaaaaaaaaaaaaaaaa", testUser1ID, testUser2ID)
	seedDMPost(t, db, postIDs(), testUser1ID, "regdmaaaaaaaaaaaaaaaaaaaaa", now)
	seedDMPost(t, db, postIDs(), testUser2ID, "regdmaaaaaaaaaaaaaaaaaaaaa", now)

	got, err := s.TopDMsForUserSince(context.Background(), testUser1ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopDMsForUserSince: %v", err)
	}
	for _, item := range got.Items {
		if item.ChannelID == "selfdmaaaaaaaaaaaaaaaaaaaa" {
			t.Fatalf("self DM leaked into top DMs: %#v", got.Items)
		}
	}
	found := false
	for _, item := range got.Items {
		if item.ChannelID == "regdmaaaaaaaaaaaaaaaaaaaaa" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected regular DM to appear; got %#v", got.Items)
	}
}

func TestStore_TopDMsForUserSince_excludesBotDM(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000
	postIDs := postIDGen('b')

	botID := "ubotaaaaaaaaaaaaaaaaaaaaaa"
	mustExec(t, db, `INSERT INTO bots (userid, deleteat) VALUES ($1, 0)`, botID)
	seedDMChannel(t, db, "botdmaaaaaaaaaaaaaaaaaaaaa", testUser1ID, botID)
	for range 5 {
		seedDMPost(t, db, postIDs(), testUser1ID, "botdmaaaaaaaaaaaaaaaaaaaaa", now)
	}

	// Regular DM.
	seedDMChannel(t, db, "humdmaaaaaaaaaaaaaaaaaaaaa", testUser1ID, testUser2ID)
	seedDMPost(t, db, postIDs(), testUser2ID, "humdmaaaaaaaaaaaaaaaaaaaaa", now)

	got, err := s.TopDMsForUserSince(context.Background(), testUser1ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopDMsForUserSince: %v", err)
	}
	for _, item := range got.Items {
		if item.ChannelID == "botdmaaaaaaaaaaaaaaaaaaaaa" {
			t.Fatalf("bot DM leaked into top DMs: %#v", got.Items)
		}
	}
	if len(got.Items) != 1 || got.Items[0].ChannelID != "humdmaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("expected only the human DM; got %#v", got.Items)
	}
}

func TestStore_TopDMsForUserSince_participantsContainBothUsers(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	seedDMChannel(t, db, "dmaaaaaaaaaaaaaaaaaaaaaaaa", testUser1ID, testUser2ID)
	seedDMPost(t, db, "paaaaaaaaaaaaaaaaaaaaaaaaa", testUser1ID, "dmaaaaaaaaaaaaaaaaaaaaaaaa", now)
	seedDMPost(t, db, "pbbbbbbbbbbbbbbbbbbbbbbbbb", testUser2ID, "dmaaaaaaaaaaaaaaaaaaaaaaaa", now)

	got, err := s.TopDMsForUserSince(context.Background(), testUser1ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopDMsForUserSince: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 DM; got %#v", got.Items)
	}
	parts := splitParticipants(got.Items[0].Participants)
	sort.Strings(parts)
	want := []string{testUser1ID, testUser2ID}
	sort.Strings(want)
	if !equalStringSlices(parts, want) {
		t.Errorf("participants = %v; want %v", parts, want)
	}
}

func TestStore_OutgoingDMCounts(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000
	postIDs := postIDGen('o')

	seedDMChannel(t, db, "dm0aaaaaaaaaaaaaaaaaaaaaaa", testUser1ID, testUser2ID)
	seedDMChannel(t, db, "dm1aaaaaaaaaaaaaaaaaaaaaaa", testUser1ID, testUser3ID)

	for range 3 {
		seedDMPost(t, db, postIDs(), testUser1ID, "dm0aaaaaaaaaaaaaaaaaaaaaaa", now)
	}
	for range 5 {
		seedDMPost(t, db, postIDs(), testUser2ID, "dm0aaaaaaaaaaaaaaaaaaaaaaa", now)
	}
	for range 2 {
		seedDMPost(t, db, postIDs(), testUser1ID, "dm1aaaaaaaaaaaaaaaaaaaaaaa", now)
	}

	counts, err := s.OutgoingDMCounts(context.Background(), testUser1ID, []string{"dm0aaaaaaaaaaaaaaaaaaaaaaa", "dm1aaaaaaaaaaaaaaaaaaaaaaa"}, since)
	if err != nil {
		t.Fatalf("OutgoingDMCounts: %v", err)
	}
	if counts["dm0aaaaaaaaaaaaaaaaaaaaaaa"] != 3 {
		t.Errorf("dm0 outgoing = %d; want 3", counts["dm0aaaaaaaaaaaaaaaaaaaaaaa"])
	}
	if counts["dm1aaaaaaaaaaaaaaaaaaaaaaa"] != 2 {
		t.Errorf("dm1 outgoing = %d; want 2", counts["dm1aaaaaaaaaaaaaaaaaaaaaaa"])
	}
}

// splitParticipants splits the comma-joined participants string the store
// returns into a slice. Equivalent to strings.Split but local to keep the
// test surface minimal.
func splitParticipants(s string) []string {
	if s == "" {
		return nil
	}
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
