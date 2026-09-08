package insights

import (
	"testing"
	"time"
)

func startOfTodayUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func TestStartOfWindowUTC(t *testing.T) {
	startOfToday := startOfTodayUTC()

	cases := []struct {
		name      string
		timeRange string
		want      time.Time
	}{
		{"7_day is the seven days preceding today", TimeRange7Day, startOfToday.AddDate(0, 0, -7)},
		{"28_day is the twenty-eight days preceding today", TimeRange28Day, startOfToday.AddDate(0, 0, -28)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := StartOfWindowUTC(tc.timeRange)
			if err != nil {
				t.Fatalf("StartOfWindowUTC(%q): %v", tc.timeRange, err)
			}
			if !got.Equal(tc.want) {
				t.Fatalf("got %v; want %v", got, tc.want)
			}
		})
	}
}

// The window must land on midnight UTC regardless of the local clock, because
// one cached snapshot is shared by every member of the team. A window that
// depended on the caller's timezone would be correct only for whoever
// happened to populate the entry.
func TestStartOfWindowUTC_isMidnightUTC(t *testing.T) {
	for _, tr := range []string{TimeRange7Day, TimeRange28Day} {
		got, err := StartOfWindowUTC(tr)
		if err != nil {
			t.Fatalf("StartOfWindowUTC(%q): %v", tr, err)
		}
		if got.Location() != time.UTC {
			t.Errorf("%s: location = %v; want UTC", tr, got.Location())
		}
		if h, m, s := got.Clock(); h != 0 || m != 0 || s != 0 {
			t.Errorf("%s: clock = %02d:%02d:%02d; want midnight", tr, h, m, s)
		}
	}
}

// "today" was dropped along with the move to a daily snapshot: a snapshot
// rebuilt once a day cannot answer a since-midnight question.
func TestStartOfWindowUTC_rejectsToday(t *testing.T) {
	if _, err := StartOfWindowUTC("today"); err == nil {
		t.Fatal("expected 'today' to be rejected; got nil error")
	}
}

func TestStartOfWindowUTC_invalid(t *testing.T) {
	if _, err := StartOfWindowUTC("7_days"); err == nil {
		t.Fatal("expected error for invalid time range; got nil")
	}
}

func TestNumberOfDaysForTimeRange(t *testing.T) {
	cases := map[string]int{
		TimeRange7Day:  7,
		TimeRange28Day: 28,
		"today":        0,
		"invalid":      0,
	}
	for in, want := range cases {
		if got := NumberOfDaysForTimeRange(in); got != want {
			t.Errorf("NumberOfDaysForTimeRange(%q) = %d; want %d", in, got, want)
		}
	}
}
