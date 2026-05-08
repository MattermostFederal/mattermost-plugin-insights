package store

import (
	"context"
	"database/sql"
	"testing"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// Fixture pattern adapted from server/channels/api4/insights_test.go's
// TestGetTopThreadsForTeamSince in commit 26617fcbdc:
//
//	one public channel and one private channel on the same team;
//	one thread in each;
//	public thread:  user1 root + user2 reply  -> ReplyCount 1
//	private thread: user1 root + user1 reply x2 -> ReplyCount 2
//
// The original assertions:
//   - team scope as user1 (member of both channels): 2 threads, private first
//   - team scope as user2 (member of public only):  1 thread (public)
//   - team scope as user2 after adding to private:  2 threads

const (
	testThreadPublicRoot  = "rootpubaaaaaaaaaaaaaaaaaa"
	testThreadPrivateRoot = "rootprvaaaaaaaaaaaaaaaaaa"
)

type threadFixture struct {
	publicChan, privateChan string
	publicRoot, privateRoot string
}

func seedThreadFixture(t *testing.T, db *sql.DB) threadFixture {
	t.Helper()
	now := nowMillis()

	mustExec(t, db, `INSERT INTO channels (id, type, teamid) VALUES ($1, 'O', $2)`, testPublicChanID, testTeamID)
	mustExec(t, db, `INSERT INTO publicchannels (id, teamid) VALUES ($1, $2)`, testPublicChanID, testTeamID)
	mustExec(t, db, `INSERT INTO channels (id, type, teamid) VALUES ($1, 'P', $2)`, testPrivateChanID, testTeamID)

	// memberships: user1 is in both, user2 only in public for now.
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, testPublicChanID, testUser1ID)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, testPublicChanID, testUser2ID)
	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, testPrivateChanID, testUser1ID)

	// public thread: user1 root + user2 reply (reply_count=1)
	mustExec(t, db, `INSERT INTO posts (id, userid, channelid, createat) VALUES ($1, $2, $3, $4)`, testThreadPublicRoot, testUser1ID, testPublicChanID, now)
	mustExec(t, db, `INSERT INTO threads (postid, channelid, replycount, lastreplyat, participants) VALUES ($1, $2, 1, $3, $4)`,
		testThreadPublicRoot, testPublicChanID, now, `["`+testUser1ID+`","`+testUser2ID+`"]`)

	// private thread: user1 root + user1 replies x2 (reply_count=2)
	mustExec(t, db, `INSERT INTO posts (id, userid, channelid, createat) VALUES ($1, $2, $3, $4)`, testThreadPrivateRoot, testUser1ID, testPrivateChanID, now)
	mustExec(t, db, `INSERT INTO threads (postid, channelid, replycount, lastreplyat, participants) VALUES ($1, $2, 2, $3, $4)`,
		testThreadPrivateRoot, testPrivateChanID, now, `["`+testUser1ID+`"]`)

	return threadFixture{
		publicChan:  testPublicChanID,
		privateChan: testPrivateChanID,
		publicRoot:  testThreadPublicRoot,
		privateRoot: testThreadPrivateRoot,
	}
}

func TestStore_TopThreadsForTeamSince_user1SeesBoth(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	fix := seedThreadFixture(t, db)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopThreadsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopThreadsForTeamSince(user1): %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("user1 should see 2 threads, got %d: %#v", len(got.Items), got.Items)
	}
	if got.Items[0].PostID != fix.privateRoot {
		t.Errorf("first thread = %s; want private root %s", got.Items[0].PostID, fix.privateRoot)
	}
	if got.Items[1].PostID != fix.publicRoot {
		t.Errorf("second thread = %s; want public root %s", got.Items[1].PostID, fix.publicRoot)
	}
	if got.Items[0].UserID != testUser1ID {
		t.Errorf("private thread root userid = %s; want %s", got.Items[0].UserID, testUser1ID)
	}
}

func TestStore_TopThreadsForTeamSince_user2SeesOnlyPublic(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	fix := seedThreadFixture(t, db)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopThreadsForTeamSince(context.Background(), testTeamID, testUser2ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopThreadsForTeamSince(user2): %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("user2 should see 1 thread (public only), got %d: %#v", len(got.Items), got.Items)
	}
	if got.Items[0].PostID != fix.publicRoot {
		t.Errorf("thread = %s; want public root %s", got.Items[0].PostID, fix.publicRoot)
	}
}

func TestStore_TopThreadsForTeamSince_user2SeesBothAfterPrivateMembership(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	seedThreadFixture(t, db)

	mustExec(t, db, `INSERT INTO channelmembers (channelid, userid) VALUES ($1, $2)`, testPrivateChanID, testUser2ID)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopThreadsForTeamSince(context.Background(), testTeamID, testUser2ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopThreadsForTeamSince: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("user2 should see 2 threads after joining private, got %d: %#v", len(got.Items), got.Items)
	}
}

func TestStore_TopThreadsForTeamSince_excludesThreadsBeforeWindow(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	seedThreadFixture(t, db)

	// since = 1 hour from now -> nothing in the fixture qualifies.
	since := nowMillis() + 60*60*1000

	got, err := s.TopThreadsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopThreadsForTeamSince: %v", err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("expected no threads before window; got %#v", got.Items)
	}
}

func TestStore_TopThreadsForTeamSince_excludesDeletedThreads(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	seedThreadFixture(t, db)

	mustExec(t, db, `UPDATE threads SET threaddeleteat = $1 WHERE postid = $2`, nowMillis(), testThreadPublicRoot)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopThreadsForTeamSince(context.Background(), testTeamID, testUser1ID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopThreadsForTeamSince: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected only private thread (public was soft-deleted); got %#v", got.Items)
	}
	if got.Items[0].PostID != testThreadPrivateRoot {
		t.Errorf("survivor = %s; want %s", got.Items[0].PostID, testThreadPrivateRoot)
	}
}

func TestStore_TopThreadsForUserSince_onlyFollowedThreads(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	seedThreadFixture(t, db)

	// user1 follows only the private thread; the public thread should be
	// excluded even though user1 is a member of the public channel.
	mustExec(t, db, `INSERT INTO threadmemberships (postid, userid, following) VALUES ($1, $2, TRUE)`, testThreadPrivateRoot, testUser1ID)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopThreadsForUserSince(context.Background(), testUser1ID, testTeamID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopThreadsForUserSince: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 followed thread; got %#v", got.Items)
	}
	if got.Items[0].PostID != testThreadPrivateRoot {
		t.Errorf("followed thread = %s; want %s", got.Items[0].PostID, testThreadPrivateRoot)
	}
}

func TestStore_TopThreadsForUserSince_followingFalseExcluded(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)
	seedThreadFixture(t, db)

	mustExec(t, db, `INSERT INTO threadmemberships (postid, userid, following) VALUES ($1, $2, FALSE)`, testThreadPrivateRoot, testUser1ID)

	since := nowMillis() - 24*60*60*1000

	got, err := s.TopThreadsForUserSince(context.Background(), testUser1ID, testTeamID, since, 0, 10)
	if err != nil {
		t.Fatalf("TopThreadsForUserSince: %v", err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("Following=FALSE should be excluded; got %#v", got.Items)
	}
}
