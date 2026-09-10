package store

import (
	"context"
	"database/sql"
	"sort"
	"testing"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

// Fixture — testUser1ID is a member of two private channels in the team and
// one in another team, plus a public channel. testUser2ID is a member of only
// one of the team's private channels. A third private channel in the team has
// no members at all, and a fourth is soft-deleted.

const (
	visPrivMemberA  = "vis0aaaaaaaaaaaaaaaaaaaaaa"
	visPrivMemberB  = "vis1aaaaaaaaaaaaaaaaaaaaaa"
	visPrivNotMine  = "vis2aaaaaaaaaaaaaaaaaaaaaa"
	visPrivDeleted  = "vis3aaaaaaaaaaaaaaaaaaaaaa"
	visPublicMember = "vis4aaaaaaaaaaaaaaaaaaaaaa"
	visPrivOther    = "vis5aaaaaaaaaaaaaaaaaaaaaa"
)

func seedVisibilityFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	seedActivityChannel(t, db, visPrivMemberA, "P", testTeamID, "priv-a", "", 100, 0, 0)
	seedMembers(t, db, visPrivMemberA, testUser1ID, testUser2ID)

	seedActivityChannel(t, db, visPrivMemberB, "P", testTeamID, "priv-b", "", 100, 0, 0)
	seedMembers(t, db, visPrivMemberB, testUser1ID)

	// Private, in the team, but user1 is not a member.
	seedActivityChannel(t, db, visPrivNotMine, "P", testTeamID, "priv-notmine", "", 100, 0, 0)
	seedMembers(t, db, visPrivNotMine, testUser3ID)

	// Private and user1 is a member, but the channel is archived.
	seedActivityChannel(t, db, visPrivDeleted, "P", testTeamID, "priv-deleted", "", 100, 0, 999)
	seedMembers(t, db, visPrivDeleted, testUser1ID)

	// Public: visible to the whole team, so it must not be enumerated here
	// even though user1 holds a membership row.
	seedActivityChannel(t, db, visPublicMember, "O", testTeamID, "pub", "", 100, 0, 0)
	seedMembers(t, db, visPublicMember, testUser1ID)

	// Private and user1 is a member, but it belongs to another team.
	seedActivityChannel(t, db, visPrivOther, "P", otherTeamID, "priv-other", "", 100, 0, 0)
	seedMembers(t, db, visPrivOther, testUser1ID)
}

func privateIDs(t *testing.T, db *sql.DB, userID string) []string {
	t.Helper()
	got, err := NewFromDB(db).PrivateChannelIDsForUser(context.Background(), userID, testTeamID)
	if err != nil {
		t.Fatalf("PrivateChannelIDsForUser: %v", err)
	}
	sort.Strings(got)
	return got
}

func TestStore_PrivateChannelIDsForUser_returnsOnlyOwnTeamPrivateMemberships(t *testing.T) {
	db := storetest.NewDB(t)
	seedVisibilityFixture(t, db)

	want := []string{visPrivMemberA, visPrivMemberB}
	sort.Strings(want)

	got := privateIDs(t, db, testUser1ID)
	if len(got) != len(want) {
		t.Fatalf("got %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v; want %v", got, want)
		}
	}
}

// Two members of the same team must get different sets — that difference is
// the whole reason the filter is applied per request rather than baked into
// the cached aggregate.
func TestStore_PrivateChannelIDsForUser_differsPerUser(t *testing.T) {
	db := storetest.NewDB(t)
	seedVisibilityFixture(t, db)

	u1 := privateIDs(t, db, testUser1ID)
	u2 := privateIDs(t, db, testUser2ID)

	if len(u1) != 2 {
		t.Errorf("user1 got %v; want 2 channels", u1)
	}
	if len(u2) != 1 || u2[0] != visPrivMemberA {
		t.Errorf("user2 got %v; want only %s", u2, visPrivMemberA)
	}
}

func TestStore_PrivateChannelIDsForUser_emptyForUserWithNoPrivateChannels(t *testing.T) {
	db := storetest.NewDB(t)
	seedVisibilityFixture(t, db)

	got := privateIDs(t, db, "nobodyaaaaaaaaaaaaaaaaaaaa")
	if len(got) != 0 {
		t.Errorf("got %v; want empty", got)
	}
}
