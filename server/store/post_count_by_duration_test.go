package store

import (
	"context"
	"testing"
	"time"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/store/storetest"
)

func TestStore_PostCountsByDuration_emptyChannelIDs(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	got, err := s.PostCountsByDuration(context.Background(), nil, 0, "", insights.PostsByDay, "UTC")
	if err != nil {
		t.Fatalf("PostCountsByDuration: %v", err)
	}
	if len(got) > 0 {
		t.Fatalf("expected empty result for empty channel ids; got %#v", got)
	}
}

func TestStore_PostCountsByDuration_groupsByDay(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	postIDs := postIDGen('y')

	mustExec(t, db, `INSERT INTO channels (id, type, teamid, name) VALUES ('chDayaaaaaaaaaaaaaaaaaaaaa', 'O', $1, 'day-channel')`, testTeamID)

	// Three posts on the same day, two on the prior day. Day boundaries are
	// in UTC because we pass "UTC" as the location.
	day0 := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC).UnixMilli()
	day1 := time.Date(2024, 1, 11, 8, 0, 0, 0, time.UTC).UnixMilli()
	day1Other := time.Date(2024, 1, 11, 23, 0, 0, 0, time.UTC).UnixMilli()

	for _, ts := range []int64{day0, day0, day1, day1, day1Other} {
		mustExec(t, db,
			`INSERT INTO posts (id, userid, channelid, createat, type, deleteat) VALUES ($1, $2, 'chDayaaaaaaaaaaaaaaaaaaaaa', $3, '', 0)`,
			postIDs(), testUser1ID, ts)
	}

	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	got, err := s.PostCountsByDuration(context.Background(), []string{"chDayaaaaaaaaaaaaaaaaaaaaa"}, since, "", insights.PostsByDay, "UTC")
	if err != nil {
		t.Fatalf("PostCountsByDuration: %v", err)
	}
	byDuration := map[string]int{}
	for _, row := range got {
		byDuration[row.Duration] = row.PostCount
	}
	if byDuration["2024-01-10"] != 2 {
		t.Errorf("2024-01-10 count = %d; want 2 (got %#v)", byDuration["2024-01-10"], got)
	}
	if byDuration["2024-01-11"] != 3 {
		t.Errorf("2024-01-11 count = %d; want 3 (got %#v)", byDuration["2024-01-11"], got)
	}
}

func TestStore_PostCountsByDuration_groupsByHour(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	postIDs := postIDGen('h')

	mustExec(t, db, `INSERT INTO channels (id, type, teamid, name) VALUES ('chHouraaaaaaaaaaaaaaaaaaaa', 'O', $1, 'hour-channel')`, testTeamID)

	// Two posts at 10:30, one at 10:55 (same hour bucket), one at 11:05.
	t1 := time.Date(2024, 1, 10, 10, 30, 0, 0, time.UTC).UnixMilli()
	t2 := time.Date(2024, 1, 10, 10, 55, 0, 0, time.UTC).UnixMilli()
	t3 := time.Date(2024, 1, 10, 11, 5, 0, 0, time.UTC).UnixMilli()
	for _, ts := range []int64{t1, t1, t2, t3} {
		mustExec(t, db,
			`INSERT INTO posts (id, userid, channelid, createat, type, deleteat) VALUES ($1, $2, 'chHouraaaaaaaaaaaaaaaaaaaa', $3, '', 0)`,
			postIDs(), testUser1ID, ts)
	}

	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	got, err := s.PostCountsByDuration(context.Background(), []string{"chHouraaaaaaaaaaaaaaaaaaaa"}, since, "", insights.PostsByHour, "UTC")
	if err != nil {
		t.Fatalf("PostCountsByDuration: %v", err)
	}
	byDuration := map[string]int{}
	for _, row := range got {
		byDuration[row.Duration] = row.PostCount
	}
	if byDuration["2024-01-10T10"] != 3 {
		t.Errorf("10:00 hour count = %d; want 3 (got %#v)", byDuration["2024-01-10T10"], got)
	}
	if byDuration["2024-01-10T11"] != 1 {
		t.Errorf("11:00 hour count = %d; want 1 (got %#v)", byDuration["2024-01-10T11"], got)
	}
}

func TestStore_PostCountsByDuration_userFilter(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	postIDs := postIDGen('u')

	mustExec(t, db, `INSERT INTO channels (id, type, teamid, name) VALUES ('chUseraaaaaaaaaaaaaaaaaaaa', 'O', $1, 'user-channel')`, testTeamID)

	day0 := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC).UnixMilli()
	mustExec(t, db,
		`INSERT INTO posts (id, userid, channelid, createat, type, deleteat) VALUES ($1, $2, 'chUseraaaaaaaaaaaaaaaaaaaa', $3, '', 0)`,
		postIDs(), testUser1ID, day0)
	mustExec(t, db,
		`INSERT INTO posts (id, userid, channelid, createat, type, deleteat) VALUES ($1, $2, 'chUseraaaaaaaaaaaaaaaaaaaa', $3, '', 0)`,
		postIDs(), testUser2ID, day0)

	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	got, err := s.PostCountsByDuration(context.Background(), []string{"chUseraaaaaaaaaaaaaaaaaaaa"}, since, testUser1ID, insights.PostsByDay, "UTC")
	if err != nil {
		t.Fatalf("PostCountsByDuration: %v", err)
	}
	if len(got) != 1 || got[0].PostCount != 1 {
		t.Errorf("expected 1 row with count=1 (only user1's post); got %#v", got)
	}
}

func TestStore_PostCountsByDuration_excludesBotPosts(t *testing.T) {
	db := storetest.NewDB(t)
	s := NewFromDB(db)

	postIDs := postIDGen('z')

	mustExec(t, db, `INSERT INTO channels (id, type, teamid, name) VALUES ('chBotaaaaaaaaaaaaaaaaaaaaa', 'O', $1, 'bot-channel')`, testTeamID)

	day0 := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC).UnixMilli()
	mustExec(t, db,
		`INSERT INTO posts (id, userid, channelid, createat, type, deleteat, props) VALUES ($1, $2, 'chBotaaaaaaaaaaaaaaaaaaaaa', $3, '', 0, $4)`,
		postIDs(), testUser2ID, day0, `{"from_bot":"true"}`)
	mustExec(t, db,
		`INSERT INTO posts (id, userid, channelid, createat, type, deleteat) VALUES ($1, $2, 'chBotaaaaaaaaaaaaaaaaaaaaa', $3, '', 0)`,
		postIDs(), testUser1ID, day0)

	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	got, err := s.PostCountsByDuration(context.Background(), []string{"chBotaaaaaaaaaaaaaaaaaaaaa"}, since, "", insights.PostsByDay, "UTC")
	if err != nil {
		t.Fatalf("PostCountsByDuration: %v", err)
	}
	if len(got) != 1 || got[0].PostCount != 1 {
		t.Errorf("expected only the human post; got %#v", got)
	}
}
