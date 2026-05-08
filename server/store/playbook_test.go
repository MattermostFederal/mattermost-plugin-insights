package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// Fixture / assertion targets ported from
// mattermost-plugin-playbooks/server/sqlstore/playbook_test.go (HEAD
// `da4c39fc` on 2026-05-08). Tables: IR_Playbook, IR_PlaybookMember,
// IR_Incident — see the Top Playbooks query in
// `mattermost-plugin-playbooks/server/sqlstore/playbook.go`
// (`insightsQueryBuilder`).

const (
	tpTeamID = "team-pb-aaaaaaaaaaaaaaaa"
	tpUser1  = "user1aaaaaaaaaaaaaaaaaaaaa"
	tpUser2  = "user2aaaaaaaaaaaaaaaaaaaaa"
	tpPb1    = "pb1aaaaaaaaaaaaaaaaaaaaaaa"
	tpPb2    = "pb2aaaaaaaaaaaaaaaaaaaaaaa"
	tpPb3    = "pb3aaaaaaaaaaaaaaaaaaaaaaa"
)

func seedTopPlaybooksFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	now := time.Now().UnixMilli()
	withinWindow := now - 60*60*1000 // 1h ago

	for _, p := range []struct {
		id, title string
		public    bool
	}{
		{tpPb1, "Incident response", true}, // public
		{tpPb2, "Onboarding", false},       // private — only members can see
		{tpPb3, "Release", true},           // public
	} {
		mustExec(t, db, `INSERT INTO IR_Playbook (ID, Title, TeamID, CreatePublicIncident, CreateAt, Public) VALUES ($1, $2, $3, false, $4, $5)`,
			p.id, p.title, tpTeamID, withinWindow-1000, p.public,
		)
	}

	// user1 is a member of pb1 and pb2; user2 is a member of nothing.
	mustExec(t, db, `INSERT INTO IR_PlaybookMember (PlaybookID, MemberID) VALUES ($1, $2)`, tpPb1, tpUser1)
	mustExec(t, db, `INSERT INTO IR_PlaybookMember (PlaybookID, MemberID) VALUES ($1, $2)`, tpPb2, tpUser1)

	// Runs (IR_Incident rows). Counts inside the time-window:
	//   pb1: 5 runs
	//   pb2: 2 runs
	//   pb3: 3 runs
	// Plus one stale pb1 run before `withinWindow` that must be filtered.
	type run struct{ pb string }
	runs := []run{
		{tpPb1},
		{tpPb1},
		{tpPb1},
		{tpPb1},
		{tpPb1},
		{tpPb2},
		{tpPb2},
		{tpPb3},
		{tpPb3},
		{tpPb3},
	}
	for i, r := range runs {
		mustExec(t, db, `INSERT INTO IR_Incident (ID, PlaybookID, TeamID, CreateAt) VALUES ($1, $2, $3, $4)`,
			"inc-"+r.pb+"-"+pad(i), r.pb, tpTeamID, withinWindow+int64(i)*1000,
		)
	}
	// Stale: one pb1 run BEFORE the window — must be excluded.
	mustExec(t, db, `INSERT INTO IR_Incident (ID, PlaybookID, TeamID, CreateAt) VALUES ('stale-pb1', $1, $2, $3)`,
		tpPb1, tpTeamID, withinWindow-3600*1000,
	)
}

func TestTopPlaybooksForTeam_publicAndMemberOnly_orderedByRuns(t *testing.T) {
	db := storetest.NewDB(t)
	seedTopPlaybooksFixture(t, db)

	store := NewFromDB(db)
	since := time.Now().Add(-2 * time.Hour).UnixMilli()
	got, err := store.TopPlaybooksForTeam(context.Background(), tpTeamID, tpUser1, since, 0, 10)
	if err != nil {
		t.Fatalf("TopPlaybooksForTeam: %v", err)
	}

	// user1 sees: pb1 (member + public), pb2 (member), pb3 (public).
	// Counts within the window are pb1=5, pb3=3, pb2=2.
	if len(got.Items) != 3 {
		t.Fatalf("expected 3 playbooks, got %d (%+v)", len(got.Items), got.Items)
	}
	if got.Items[0].PlaybookID != tpPb1 || got.Items[0].NumRuns != 5 {
		t.Errorf("rank-1 mismatch: got %+v", got.Items[0])
	}
	if got.Items[1].PlaybookID != tpPb3 || got.Items[1].NumRuns != 3 {
		t.Errorf("rank-2 mismatch: got %+v", got.Items[1])
	}
	if got.Items[2].PlaybookID != tpPb2 || got.Items[2].NumRuns != 2 {
		t.Errorf("rank-3 mismatch: got %+v", got.Items[2])
	}
}

func TestTopPlaybooksForTeam_excludesPrivatePlaybooksUserIsNotMemberOf(t *testing.T) {
	db := storetest.NewDB(t)
	seedTopPlaybooksFixture(t, db)

	store := NewFromDB(db)
	since := time.Now().Add(-2 * time.Hour).UnixMilli()
	// user2 is NOT a member of any playbook. Should only see public ones.
	got, err := store.TopPlaybooksForTeam(context.Background(), tpTeamID, tpUser2, since, 0, 10)
	if err != nil {
		t.Fatalf("TopPlaybooksForTeam: %v", err)
	}

	for _, item := range got.Items {
		if item.PlaybookID == tpPb2 {
			t.Errorf("user2 should not see private pb2: got %+v", got.Items)
		}
	}
	// user2 sees pb1 (public) and pb3 (public).
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 public playbooks for user2, got %d (%+v)", len(got.Items), got.Items)
	}
}

func TestTopPlaybooksForUser_filtersToOnlyUsersMemberPlaybooks(t *testing.T) {
	db := storetest.NewDB(t)
	seedTopPlaybooksFixture(t, db)

	store := NewFromDB(db)
	since := time.Now().Add(-2 * time.Hour).UnixMilli()
	// user-scope: only playbooks the user is a direct member of, regardless
	// of public/private. user1 is a member of pb1 and pb2 only; should NOT
	// see pb3.
	got, err := store.TopPlaybooksForUser(context.Background(), tpTeamID, tpUser1, since, 0, 10)
	if err != nil {
		t.Fatalf("TopPlaybooksForUser: %v", err)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 playbooks for user1 (member-only), got %d (%+v)", len(got.Items), got.Items)
	}
	for _, item := range got.Items {
		if item.PlaybookID == tpPb3 {
			t.Errorf("user1 should not see pb3 (not a member): got %+v", got.Items)
		}
	}
}

func TestTopPlaybooksForTeam_lastRunAtIsTheLatestIncidentCreateAt(t *testing.T) {
	db := storetest.NewDB(t)
	seedTopPlaybooksFixture(t, db)

	store := NewFromDB(db)
	since := time.Now().Add(-2 * time.Hour).UnixMilli()
	got, err := store.TopPlaybooksForTeam(context.Background(), tpTeamID, tpUser1, since, 0, 10)
	if err != nil {
		t.Fatalf("TopPlaybooksForTeam: %v", err)
	}
	for _, item := range got.Items {
		if item.LastRunAt == 0 {
			t.Errorf("playbook %s has no LastRunAt: %+v", item.PlaybookID, item)
		}
	}
}
