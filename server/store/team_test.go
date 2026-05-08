package store

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// Fixtures adapted from server/channels/api4/insights_test.go's
// TestNewTeamMembersSince in commit 26617fcbdc:
//   - members are returned with full user fields
//   - bots are excluded
//   - deleted users are excluded
//   - deleted team-membership rows are excluded
//   - results are paginated with TotalCount tracking the entire qualifying
//     set regardless of page

func seedUser(t *testing.T, db *sql.DB, id, username, firstName, lastName, position, nickname string, lastPictureUpdate int64) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO users (id, username, firstname, lastname, position, nickname, lastpictureupdate, deleteat) VALUES ($1, $2, $3, $4, $5, $6, $7, 0)`,
		id, username, firstName, lastName, position, nickname, lastPictureUpdate)
}

func seedTeamMember(t *testing.T, db *sql.DB, teamID, userID string, createAt, deleteAt int64) {
	t.Helper()
	mustExec(t, db,
		`INSERT INTO teammembers (teamid, userid, createat, deleteat) VALUES ($1, $2, $3, $4)`,
		teamID, userID, createAt, deleteAt)
}

func TestStore_NewTeamMembersSince_basicResponse(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	seedUser(t, db, "uaaaaaaaaaaaaaaaaaaaaaaaaa", "alice", "Alice", "Anders", "Eng", "ali", 1700)
	seedTeamMember(t, db, testTeamID, "uaaaaaaaaaaaaaaaaaaaaaaaaa", now, 0)

	got, err := s.NewTeamMembersSince(context.Background(), testTeamID, since, 0, 10, true)
	if err != nil {
		t.Fatalf("NewTeamMembersSince: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d; want 1", got.TotalCount)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 item; got %#v", got.Items)
	}
	item := got.Items[0]
	if item.Username != "alice" || item.FirstName != "Alice" || item.LastName != "Anders" || item.Position != "Eng" || item.Nickname != "ali" || item.LastPictureUpdate != 1700 || item.CreateAt != now {
		t.Errorf("item = %#v", item)
	}
}

func TestStore_NewTeamMembersSince_excludesBots(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	seedUser(t, db, "uhumanaaaaaaaaaaaaaaaaaaaa", "human", "", "", "", "", 0)
	seedUser(t, db, "ubotaaaaaaaaaaaaaaaaaaaaaa", "bot.user", "", "", "", "", 0)
	mustExec(t, db, `INSERT INTO bots (userid, deleteat) VALUES ($1, 0)`, "ubotaaaaaaaaaaaaaaaaaaaaaa")
	seedTeamMember(t, db, testTeamID, "uhumanaaaaaaaaaaaaaaaaaaaa", now, 0)
	seedTeamMember(t, db, testTeamID, "ubotaaaaaaaaaaaaaaaaaaaaaa", now, 0)

	got, err := s.NewTeamMembersSince(context.Background(), testTeamID, since, 0, 10, true)
	if err != nil {
		t.Fatalf("NewTeamMembersSince: %v", err)
	}
	if got.TotalCount != 1 {
		t.Errorf("TotalCount = %d; want 1 (bot excluded)", got.TotalCount)
	}
	if len(got.Items) != 1 || got.Items[0].Username != "human" {
		t.Fatalf("expected only human; got %#v", got.Items)
	}
}

func TestStore_NewTeamMembersSince_excludesDeletedUsers(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	mustExec(t, db,
		`INSERT INTO users (id, username, deleteat) VALUES ($1, $2, $3)`,
		"udelaaaaaaaaaaaaaaaaaaaaaa", "deleted", now-1000)
	seedUser(t, db, "uactiveaaaaaaaaaaaaaaaaaaa", "active", "", "", "", "", 0)
	seedTeamMember(t, db, testTeamID, "udelaaaaaaaaaaaaaaaaaaaaaa", now, 0)
	seedTeamMember(t, db, testTeamID, "uactiveaaaaaaaaaaaaaaaaaaa", now, 0)

	got, err := s.NewTeamMembersSince(context.Background(), testTeamID, since, 0, 10, true)
	if err != nil {
		t.Fatalf("NewTeamMembersSince: %v", err)
	}
	if got.TotalCount != 1 || len(got.Items) != 1 || got.Items[0].Username != "active" {
		t.Fatalf("got %#v", got)
	}
}

func TestStore_NewTeamMembersSince_excludesDeletedMembership(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	seedUser(t, db, "ulapsedaaaaaaaaaaaaaaaaaaa", "lapsed", "", "", "", "", 0)
	seedUser(t, db, "ucurrentaaaaaaaaaaaaaaaaaa", "current", "", "", "", "", 0)
	seedTeamMember(t, db, testTeamID, "ulapsedaaaaaaaaaaaaaaaaaaa", now, now-1000)
	seedTeamMember(t, db, testTeamID, "ucurrentaaaaaaaaaaaaaaaaaa", now, 0)

	got, err := s.NewTeamMembersSince(context.Background(), testTeamID, since, 0, 10, true)
	if err != nil {
		t.Fatalf("NewTeamMembersSince: %v", err)
	}
	if got.TotalCount != 1 || len(got.Items) != 1 || got.Items[0].Username != "current" {
		t.Fatalf("got %#v", got)
	}
}

func TestStore_NewTeamMembersSince_pagination(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	for i := range 3 {
		uid := fmt.Sprintf("u%-25s", fmt.Sprintf("user%d", i))[:26]
		seedUser(t, db, uid, fmt.Sprintf("user%d", i), "", "", "", "", 0)
		seedTeamMember(t, db, testTeamID, uid, now, 0)
	}

	page0, err := s.NewTeamMembersSince(context.Background(), testTeamID, since, 0, 2, true)
	if err != nil {
		t.Fatalf("page0: %v", err)
	}
	if page0.TotalCount != 3 || len(page0.Items) != 2 || !page0.HasNext {
		t.Fatalf("page0: got TotalCount=%d items=%d hasNext=%v; want 3/2/true", page0.TotalCount, len(page0.Items), page0.HasNext)
	}

	page1, err := s.NewTeamMembersSince(context.Background(), testTeamID, since, 1, 2, true)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if page1.TotalCount != 3 || len(page1.Items) != 1 || page1.HasNext {
		t.Fatalf("page1: got TotalCount=%d items=%d hasNext=%v; want 3/1/false", page1.TotalCount, len(page1.Items), page1.HasNext)
	}
}

func TestStore_NewTeamMembersSince_omitsFullNameWhenDisabled(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	now := nowMillis()
	since := now - 24*60*60*1000

	seedUser(t, db, "uaaaaaaaaaaaaaaaaaaaaaaaaa", "alice", "Alice", "Anders", "Eng", "ali", 0)
	seedTeamMember(t, db, testTeamID, "uaaaaaaaaaaaaaaaaaaaaaaaaa", now, 0)

	got, err := s.NewTeamMembersSince(context.Background(), testTeamID, since, 0, 10, false)
	if err != nil {
		t.Fatalf("NewTeamMembersSince: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 item; got %#v", got.Items)
	}
	if got.Items[0].FirstName != "" || got.Items[0].LastName != "" {
		t.Errorf("expected first/last name to be omitted with showFullName=false; got %#v", got.Items[0])
	}
	// other fields still populated
	if got.Items[0].Username != "alice" || got.Items[0].Position != "Eng" || got.Items[0].Nickname != "ali" {
		t.Errorf("other fields lost: %#v", got.Items[0])
	}
}
