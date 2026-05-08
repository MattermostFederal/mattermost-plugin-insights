package store

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// Fixture / assertion targets ported from
// mattermost-plugin-boards/server/services/store/storetests/board_insights.go
// at commit c8e729b6^ (the commit that deleted board insights). The
// deprecated test inserted boards / blocks_history / boards_history through
// the focalboard store API; here we insert directly into the same
// underlying tables and assert against the same expected counts.

const (
	tbTeamID  = "team-board-aaaaaaaaaaaaaa"
	tbUser1   = "user-id-1aaaaaaaaaaaaaaaa"
	tbUser2   = "user-id-2aaaaaaaaaaaaaaaa"
	tbBoard1  = "board-id-1"
	tbBoard2  = "board-id-2"
	tbBoard3  = "board-id-3"
	tbChannel = "channel-id-aaaaaaaaaaaaaa"
)

func seedTopBoardsFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	now := time.Now().UTC()
	insertAt := now.Add(-1 * time.Hour) // newer than `since`

	for _, b := range []struct {
		id, ttype string
	}{
		{tbBoard1, "O"},
		{tbBoard2, "P"},
		{tbBoard3, "O"},
	} {
		mustExec(t, db,
			`INSERT INTO focalboard_boards (id, insert_at, team_id, channel_id, created_by, type, title, icon, is_template, delete_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, '💬', false, 0)`,
			b.id, insertAt, tbTeamID, tbChannel, tbUser1, b.ttype, "Board "+b.id,
		)
	}

	// boards_history rows. Each row is a board "edit" by some user.
	// board1 edited 2 times by user1
	// board2 edited 1 time  by user1
	// board3 edited 0 times
	for i, h := range []struct {
		boardID, modifiedBy string
	}{
		{tbBoard1, tbUser1},
		{tbBoard1, tbUser1},
		{tbBoard2, tbUser1},
	} {
		mustExec(t, db,
			`INSERT INTO focalboard_boards_history (id, insert_at, team_id, channel_id, created_by, modified_by, type, title, is_template, delete_at)
			 VALUES ($1, $2, $3, $4, $5, $6, 'O', 'h', false, 0)`,
			h.boardID, insertAt.Add(time.Duration(i)*time.Second), tbTeamID, tbChannel, tbUser1, h.modifiedBy,
		)
	}

	// blocks_history rows. Each row is a block edit (cards, etc).
	// board1: 5 by user1, 2 by user2
	// board2: 4 by user1
	// board3: 3 by user1
	type blockEdit struct{ boardID, modifiedBy string }
	blocks := []blockEdit{
		{tbBoard1, tbUser1},
		{tbBoard1, tbUser1},
		{tbBoard1, tbUser1},
		{tbBoard1, tbUser1},
		{tbBoard1, tbUser1},
		{tbBoard1, tbUser2},
		{tbBoard1, tbUser2},
		{tbBoard2, tbUser1},
		{tbBoard2, tbUser1},
		{tbBoard2, tbUser1},
		{tbBoard2, tbUser1},
		{tbBoard3, tbUser1},
		{tbBoard3, tbUser1},
		{tbBoard3, tbUser1},
	}
	for i, b := range blocks {
		mustExec(t, db,
			`INSERT INTO focalboard_blocks_history (id, insert_at, board_id, modified_by, type, delete_at)
			 VALUES ($1, $2, $3, $4, 'card', 0)`,
			"block-"+b.boardID+"-"+pad(i), insertAt.Add(time.Duration(i)*time.Millisecond), b.boardID, b.modifiedBy,
		)
	}

	// One "system" block edit on board2 — must be filtered out.
	mustExec(t, db,
		`INSERT INTO focalboard_blocks_history (id, insert_at, board_id, modified_by, type, delete_at)
		 VALUES ('block-system-1', $1, $2, 'system', 'card', 0)`,
		insertAt, tbBoard2,
	)

	// Direct membership: user2 is a direct member of board2.
	mustExec(t, db,
		`INSERT INTO focalboard_board_members (board_id, user_id, scheme_admin) VALUES ($1, $2, false)`,
		tbBoard2, tbUser2,
	)
}

func pad(i int) string {
	if i < 10 {
		return "0" + string(rune('0'+i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}

func TestTopBoardsForTeam_ordersByActivityCount(t *testing.T) {
	db := storetest.NewDB(t)

	seedTopBoardsFixture(t, db)

	store := NewFromDB(db)
	since := time.Now().UTC().Add(-2 * time.Hour).UnixMilli()
	got, err := store.TopBoardsForTeam(context.Background(), tbTeamID,
		[]string{tbBoard1, tbBoard2, tbBoard3}, since, 0, 10)
	if err != nil {
		t.Fatalf("TopBoardsForTeam: %v", err)
	}
	if len(got.Items) != 3 {
		t.Fatalf("expected 3 boards, got %d (%+v)", len(got.Items), got.Items)
	}

	// Expected (board+blocks edits, system filtered out):
	//   board1: 2 boards_history + 7 blocks_history = 9
	//   board2: 1 boards_history + 4 blocks_history = 5
	//   board3: 0 boards_history + 3 blocks_history = 3
	if got.Items[0].BoardID != tbBoard1 || got.Items[0].ActivityCount != "9" {
		t.Errorf("rank-1 mismatch: got %+v", got.Items[0])
	}
	if got.Items[1].BoardID != tbBoard2 || got.Items[1].ActivityCount != "5" {
		t.Errorf("rank-2 mismatch: got %+v", got.Items[1])
	}
	if got.Items[2].BoardID != tbBoard3 || got.Items[2].ActivityCount != "3" {
		t.Errorf("rank-3 mismatch: got %+v", got.Items[2])
	}

	// active_users on board1 must contain user1 and user2 (deduped).
	users := append([]string{}, got.Items[0].ActiveUsers...)
	sort.Strings(users)
	want := []string{tbUser1, tbUser2}
	if strings.Join(users, ",") != strings.Join(want, ",") {
		t.Errorf("board1 active_users mismatch: got %v want %v", users, want)
	}
}

func TestTopBoardsForTeam_filtersOutDeletedBoards(t *testing.T) {
	db := storetest.NewDB(t)

	seedTopBoardsFixture(t, db)

	// Mark board1 as deleted.
	mustExec(t, db, `UPDATE focalboard_boards SET delete_at = 1 WHERE id = $1`, tbBoard1)

	store := NewFromDB(db)
	since := time.Now().UTC().Add(-2 * time.Hour).UnixMilli()
	got, err := store.TopBoardsForTeam(context.Background(), tbTeamID,
		[]string{tbBoard1, tbBoard2, tbBoard3}, since, 0, 10)
	if err != nil {
		t.Fatalf("TopBoardsForTeam: %v", err)
	}
	for _, b := range got.Items {
		if b.BoardID == tbBoard1 {
			t.Errorf("deleted board1 must not appear in results, got %+v", b)
		}
	}
}

func TestTopBoardsForTeam_excludesSystemEdits(t *testing.T) {
	db := storetest.NewDB(t)

	seedTopBoardsFixture(t, db)

	store := NewFromDB(db)
	since := time.Now().UTC().Add(-2 * time.Hour).UnixMilli()
	got, err := store.TopBoardsForTeam(context.Background(), tbTeamID,
		[]string{tbBoard1, tbBoard2, tbBoard3}, since, 0, 10)
	if err != nil {
		t.Fatalf("TopBoardsForTeam: %v", err)
	}
	// board2 should be 5 (4 blocks + 1 board edit), NOT 6 (the +1 system
	// edit must be filtered out).
	for _, b := range got.Items {
		if b.BoardID == tbBoard2 && b.ActivityCount != "5" {
			t.Errorf("system edit not filtered out: board2 got count %s, want 5", b.ActivityCount)
		}
	}
}

func TestTopBoardsForUser_filtersToBoardsCreatedByOrParticipatedIn(t *testing.T) {
	db := storetest.NewDB(t)

	seedTopBoardsFixture(t, db)

	store := NewFromDB(db)
	since := time.Now().UTC().Add(-2 * time.Hour).UnixMilli()
	// user2 only participated in board1 (via blocks_history) and is a
	// direct member of board2. user2 didn't create any boards.
	got, err := store.TopBoardsForUser(context.Background(), tbTeamID, tbUser2,
		[]string{tbBoard1, tbBoard2}, since, 0, 10)
	if err != nil {
		t.Fatalf("TopBoardsForUser: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 board for user2 (only board1 has user2 in active_users), got %d", len(got.Items))
	}
	if got.Items[0].BoardID != tbBoard1 {
		t.Errorf("user2 should see board1 (participated via blocks_history), got %s", got.Items[0].BoardID)
	}
}
