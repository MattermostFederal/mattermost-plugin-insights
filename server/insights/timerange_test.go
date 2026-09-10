package insights

import (
	"testing"
	"time"
)

func startOfTodayUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func TestWindowUTC(t *testing.T) {
	startOfToday := startOfTodayUTC()

	cases := []struct {
		name      string
		timeRange string
		wantStart time.Time
	}{
		{"1_day is yesterday", TimeRange1Day, startOfToday.AddDate(0, 0, -1)},
		{"7_day is the seven days preceding today", TimeRange7Day, startOfToday.AddDate(0, 0, -7)},
		{"28_day is the twenty-eight days preceding today", TimeRange28Day, startOfToday.AddDate(0, 0, -28)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, err := WindowUTC(tc.timeRange)
			if err != nil {
				t.Fatalf("WindowUTC(%q): %v", tc.timeRange, err)
			}
			if !w.Start.Equal(tc.wantStart) {
				t.Errorf("Start = %v; want %v", w.Start, tc.wantStart)
			}
			// Every window ends at midnight today, never "now".
			if !w.End.Equal(startOfToday) {
				t.Errorf("End = %v; want %v (midnight today)", w.End, startOfToday)
			}
		})
	}
}

// The window must describe a period that has already finished. If it ran up to
// "now" instead, a cached copy would be stale the instant it was written and
// the once-daily snapshot could not work.
func TestWindowUTC_excludesToday(t *testing.T) {
	w, err := WindowUTC(TimeRange1Day)
	if err != nil {
		t.Fatalf("WindowUTC: %v", err)
	}
	if w.End.After(time.Now().UTC()) {
		t.Error("window extends into the future")
	}
	if !w.End.Equal(startOfTodayUTC()) {
		t.Errorf("End = %v; want midnight today, not now", w.End)
	}
	if got := w.End.Sub(w.Start); got != 24*time.Hour {
		t.Errorf("1_day spans %v; want exactly 24h", got)
	}
}

// The window boundary must land on midnight UTC regardless of the local clock,
// because one cached snapshot is shared by every member of the team.
func TestWindowUTC_isMidnightUTC(t *testing.T) {
	for _, tr := range []string{TimeRange1Day, TimeRange7Day, TimeRange28Day} {
		w, err := WindowUTC(tr)
		if err != nil {
			t.Fatalf("WindowUTC(%q): %v", tr, err)
		}
		for label, ts := range map[string]time.Time{"Start": w.Start, "End": w.End} {
			if ts.Location() != time.UTC {
				t.Errorf("%s %s: location = %v; want UTC", tr, label, ts.Location())
			}
			if h, m, s := ts.Clock(); h != 0 || m != 0 || s != 0 {
				t.Errorf("%s %s: clock = %02d:%02d:%02d; want midnight", tr, label, h, m, s)
			}
		}
	}
}

// The cache key carries the window's date so entries roll over at midnight
// rather than 24 hours after whenever they were built.
func TestWindow_KeyIsTheStartDate(t *testing.T) {
	w := Window{
		Start: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC),
	}
	if got, want := w.Key(), "2026-09-08"; got != want {
		t.Errorf("Key = %q; want %q", got, want)
	}
}

// The "today" range is a moving, open-ended window — the one shape a
// periodically rebuilt snapshot genuinely cannot answer.
func TestWindowUTC_rejectsToday(t *testing.T) {
	if _, err := WindowUTC("today"); err == nil {
		t.Fatal("expected 'today' to be rejected; got nil error")
	}
}

func TestWindowUTC_invalid(t *testing.T) {
	if _, err := WindowUTC("7_days"); err == nil {
		t.Fatal("expected error for invalid time range; got nil")
	}
}

func TestNumberOfDaysForTimeRange(t *testing.T) {
	cases := map[string]int{
		TimeRange1Day:  1,
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
