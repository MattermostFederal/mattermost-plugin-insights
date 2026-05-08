package store

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// Fixture data adapted from server/channels/api4/insights_test.go's
// TestGetTopReactionsForTeamSince in commit 26617fcbdc. The original test
// inserted these reactions through the Mattermost API; we insert directly
// into the same underlying tables and assert the same expected ordering.

const (
	testTeamID         = "team0aaaaaaaaaaaaaaaaaaaaa"
	testOtherTeamID    = "team1aaaaaaaaaaaaaaaaaaaaa"
	testUser1ID        = "user1aaaaaaaaaaaaaaaaaaaaa"
	testUser2ID        = "user2aaaaaaaaaaaaaaaaaaaaa"
	testUser3ID        = "user3aaaaaaaaaaaaaaaaaaaaa"
	testPublicChanID   = "pubchan0aaaaaaaaaaaaaaaaaa"
	testPrivateChanID  = "privchanaaaaaaaaaaaaaaaaaa"
	testExcludedChanID = "excluchanaaaaaaaaaaaaaaaaa"
	testOtherTeamChan  = "otrteamchanaaaaaaaaaaaaaaa"
	testDMChanID       = "dmchanaaaaaaaaaaaaaaaaaaaa"
)

func nowMillis() int64 { return time.Now().UnixMilli() }

func mustExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec %s: %v\nargs=%v", q, err, args)
	}
}

// postIDGen returns a generator that yields successive 26-character ids of
// the shape `<prefix>` + zero-padded counter. Callers in the same test that
// need to insert posts in multiple batches without collisions should pass
// distinct prefixes.
func postIDGen(prefix ...byte) func() string {
	p := byte('p')
	if len(prefix) > 0 {
		p = prefix[0]
	}
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("%c%025d", p, n)
	}
}

// seedReaction inserts a single reaction. postID must be a fresh 26-char id.
func seedReaction(t *testing.T, db *sql.DB, postID, userID, emojiName, channelID string, createAt int64) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO reactions (postid, userid, emojiname, channelid, createat, deleteat) VALUES ($1, $2, $3, $4, $5, 0)`,
		postID, userID, emojiName, channelID, createAt,
	)
}

// seedTeamReactionsFixture installs the canonical 22-reaction fixture from
// the deprecated TestGetTopReactionsForTeamSince. 21 reactions live "today"
// and 1 reaction (user1's "100") is timestamped 25 hours ago to exercise the
// time-range filter.
//
// Expected top-5 within the today window:
//
//	100: 6, joy: 5, smile: 4, sad: 3, happy: 2
//
// Expected page-1 single row: +1: 1.
func seedTeamReactionsFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	now := nowMillis()
	yesterday := now - 25*60*60*1000

	mustExec(t, db, `INSERT INTO channels (id, type, teamid) VALUES ($1, 'O', $2)`, testPublicChanID, testTeamID)
	mustExec(t, db, `INSERT INTO publicchannels (id, teamid) VALUES ($1, $2)`, testPublicChanID, testTeamID)

	id := postIDGen()
	add := func(userID, emoji string, ts int64) {
		seedReaction(t, db, id(), userID, emoji, testPublicChanID, ts)
	}

	// post1: u1+u2 happy/sad; u1 also smile/joy/100
	add(testUser1ID, "happy", now)
	add(testUser2ID, "happy", now)
	add(testUser1ID, "sad", now)
	add(testUser2ID, "sad", now)
	add(testUser1ID, "smile", now)
	add(testUser1ID, "joy", now)
	add(testUser1ID, "100", now)

	// post2: u1 sad/smile/joy/100
	add(testUser1ID, "sad", now)
	add(testUser1ID, "smile", now)
	add(testUser1ID, "joy", now)
	add(testUser1ID, "100", now)

	// post3: u1 smile/joy/100; u2 smile
	add(testUser1ID, "smile", now)
	add(testUser2ID, "smile", now)
	add(testUser1ID, "joy", now)
	add(testUser1ID, "100", now)

	// post4: u1 joy/100; u2 joy
	add(testUser1ID, "joy", now)
	add(testUser2ID, "joy", now)
	add(testUser1ID, "100", now)

	// post5: u1 100; u2 100/+1
	add(testUser1ID, "100", now)
	add(testUser2ID, "100", now)
	add(testUser2ID, "+1", now)

	// 22nd reaction: timestamped 25h ago, must be excluded by today's filter
	add(testUser1ID, "100", yesterday)
}

func TestStore_TopReactionsForTeamSince_basicOrdering(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	seedTeamReactionsFixture(t, db)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopReactionsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 5)
	if err != nil {
		t.Fatalf("TopReactionsForTeamSince: %v", err)
	}
	if got == nil || len(got.Items) != 5 {
		t.Fatalf("expected 5 items, got %#v", got)
	}

	wantNames := []string{"100", "joy", "smile", "sad", "happy"}
	wantCounts := []int64{6, 5, 4, 3, 2}
	for i, item := range got.Items {
		if item.EmojiName != wantNames[i] || item.Count != wantCounts[i] {
			t.Fatalf("rank %d: got %s=%d; want %s=%d", i, item.EmojiName, item.Count, wantNames[i], wantCounts[i])
		}
	}
	if !got.HasNext {
		t.Errorf("hasNext = false; want true (one more emoji on page 1)")
	}
}

func TestStore_TopReactionsForTeamSince_pagination(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	seedTeamReactionsFixture(t, db)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopReactionsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 1, 5)
	if err != nil {
		t.Fatalf("TopReactionsForTeamSince page=1: %v", err)
	}
	if got == nil || len(got.Items) != 1 {
		t.Fatalf("expected 1 item on page 1, got %#v", got)
	}
	if got.Items[0].EmojiName != "+1" || got.Items[0].Count != 1 {
		t.Fatalf("page 1 item = %s/%d; want +1/1", got.Items[0].EmojiName, got.Items[0].Count)
	}
	if got.HasNext {
		t.Errorf("hasNext = true on last page; want false")
	}
}

func TestStore_TopReactionsForTeamSince_excludesPrivateChannelsUserIsNotMemberOf(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	seedTeamReactionsFixture(t, db)

	now := nowMillis()
	since := now - 24*60*60*1000

	mustExec(t, db, `INSERT INTO channels (id, type, teamid) VALUES ($1, 'P', $2)`, testExcludedChanID, testTeamID)

	id := postIDGen()
	for range 10 {
		seedReaction(t, db, id(), testUser2ID, "confused", testExcludedChanID, now)
	}

	got, err := s.TopReactionsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 5)
	if err != nil {
		t.Fatalf("TopReactionsForTeamSince: %v", err)
	}
	for _, item := range got.Items {
		if item.EmojiName == "confused" {
			t.Fatalf("private channel user is not a member of leaked into team reactions: %v", got.Items)
		}
	}
}

func TestStore_TopReactionsForTeamSince_includesPrivateChannelsUserIsMemberOf(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	mustExec(t, db, `INSERT INTO channels (id, type, teamid) VALUES ($1, 'P', $2)`, testPrivateChanID, testTeamID)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, testPrivateChanID, testUser1ID)

	id := postIDGen()
	for range 7 {
		seedReaction(t, db, id(), testUser2ID, "wow", testPrivateChanID, now)
	}

	got, err := s.TopReactionsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 5)
	if err != nil {
		t.Fatalf("TopReactionsForTeamSince: %v", err)
	}
	if got == nil || len(got.Items) == 0 {
		t.Fatalf("expected items; got %#v", got)
	}
	if got.Items[0].EmojiName != "wow" || got.Items[0].Count != 7 {
		t.Fatalf("top = %s/%d; want wow/7", got.Items[0].EmojiName, got.Items[0].Count)
	}
}

func TestStore_TopReactionsForUserSince_userOnlyOwnReactions(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	seedTeamReactionsFixture(t, db)

	since := nowMillis() - 24*60*60*1000

	// User-scope without team filter: every reaction by user1 across all
	// channels. user1 contributed: happy 1, sad 2, smile 3, joy 4, 100 5.
	got, err := s.TopReactionsForUserSince(context.Background(), testUser1ID, "", since, 0, 5)
	if err != nil {
		t.Fatalf("TopReactionsForUserSince: %v", err)
	}
	if got == nil {
		t.Fatalf("nil result")
	}
	wantPairs := map[string]int64{"100": 5, "joy": 4, "smile": 3, "sad": 2, "happy": 1}
	if len(got.Items) != 5 {
		t.Fatalf("expected 5 items, got %d: %#v", len(got.Items), got.Items)
	}
	for _, item := range got.Items {
		if want, ok := wantPairs[item.EmojiName]; !ok || item.Count != want {
			t.Errorf("got %s=%d; not in expected map %v", item.EmojiName, item.Count, wantPairs)
		}
	}
}

func TestStore_TopReactionsForUserSince_excludesOldRows(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	old := now - 100*60*60*1000
	since := now - 24*60*60*1000

	mustExec(t, db, `INSERT INTO channels (id, type, teamid) VALUES ($1, 'O', $2)`, testPublicChanID, testTeamID)
	mustExec(t, db, `INSERT INTO publicchannels (id, teamid) VALUES ($1, $2)`, testPublicChanID, testTeamID)

	id := postIDGen()
	seedReaction(t, db, id(), testUser1ID, "newer", testPublicChanID, now)
	seedReaction(t, db, id(), testUser1ID, "older", testPublicChanID, old)

	got, err := s.TopReactionsForUserSince(context.Background(), testUser1ID, "", since, 0, 5)
	if err != nil {
		t.Fatalf("TopReactionsForUserSince: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].EmojiName != "newer" {
		t.Fatalf("expected only 'newer' in window; got %#v", got.Items)
	}
}

func TestStore_TopReactionsForUserSince_teamFilter(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	mustExec(t, db, `INSERT INTO channels (id, type, teamid) VALUES ($1, 'O', $2)`, testPublicChanID, testTeamID)
	mustExec(t, db, `INSERT INTO channels (id, type, teamid) VALUES ($1, 'O', $2)`, testOtherTeamChan, testOtherTeamID)
	mustExec(t, db, `INSERT INTO channels (id, type, teamid) VALUES ($1, 'D', '')`, testDMChanID)

	id := postIDGen()
	seedReaction(t, db, id(), testUser1ID, "inteam", testPublicChanID, now)
	seedReaction(t, db, id(), testUser1ID, "outteam", testOtherTeamChan, now)
	seedReaction(t, db, id(), testUser1ID, "indm", testDMChanID, now)

	got, err := s.TopReactionsForUserSince(context.Background(), testUser1ID, testTeamID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopReactionsForUserSince: %v", err)
	}
	have := map[string]bool{}
	for _, item := range got.Items {
		have[item.EmojiName] = true
	}
	if !have["inteam"] {
		t.Errorf("expected reaction in target team to be included; got %#v", got.Items)
	}
	if !have["indm"] {
		t.Errorf("expected reaction in DM to be included regardless of team; got %#v", got.Items)
	}
	if have["outteam"] {
		t.Errorf("did not expect reaction from a different team; got %#v", got.Items)
	}
}
