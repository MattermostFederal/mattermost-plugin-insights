package insights

import (
	"testing"
	"time"
)

func TestStartOfDayForTimeRange(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}

	startOfToday := time.Date(time.Now().In(loc).Year(), time.Now().In(loc).Month(), time.Now().In(loc).Day(), 0, 0, 0, 0, loc)

	cases := []struct {
		name      string
		timeRange string
		want      time.Time
	}{
		{"today", TimeRangeToday, startOfToday},
		{"7_day", TimeRange7Day, startOfToday.Add(-144 * time.Hour)},
		{"28_day", TimeRange28Day, startOfToday.Add(-648 * time.Hour)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := StartOfDayForTimeRange(tc.timeRange, loc)
			if err != nil {
				t.Fatalf("StartOfDayForTimeRange(%q): %v", tc.timeRange, err)
			}
			if !got.Equal(tc.want) {
				t.Fatalf("got %v; want %v", got, tc.want)
			}
		})
	}
}

func TestStartOfDayForTimeRange_invalid(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	if _, err := StartOfDayForTimeRange("7_days", loc); err == nil {
		t.Fatalf("expected error for invalid time range; got nil")
	}
}

func TestNumberOfDaysForTimeRange(t *testing.T) {
	cases := map[string]int{
		TimeRangeToday: 1,
		TimeRange7Day:  7,
		TimeRange28Day: 28,
		"invalid":      0,
	}
	for in, want := range cases {
		if got := NumberOfDaysForTimeRange(in); got != want {
			t.Errorf("NumberOfDaysForTimeRange(%q) = %d; want %d", in, got, want)
		}
	}
}
