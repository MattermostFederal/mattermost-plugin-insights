package insights

import (
	"errors"
	"time"
)

// ErrInvalidTimeRange is returned when a request's time_range query param is
// not one of the supported values.
var ErrInvalidTimeRange = errors.New("time_range must be one of: 1_day, 7_day, 28_day")

// Window is a closed, immutable span of complete UTC days: Start inclusive,
// End exclusive.
//
// Closedness is what lets one snapshot serve a whole day. An open-ended
// window ("since midnight N days ago, up to now") keeps moving, so a cached
// copy is stale the moment it is written. A window that ends at midnight
// today describes a period that has already finished — rebuild it at 01:00 or
// at 23:00 and you get the same answer.
type Window struct {
	Start time.Time // inclusive
	End   time.Time // exclusive
}

// StartMillis and EndMillis are the unix-millisecond bounds the store queries
// against.
func (w Window) StartMillis() int64 { return w.Start.UnixMilli() }
func (w Window) EndMillis() int64   { return w.End.UnixMilli() }

// Key returns a stable identifier for the window, used as part of the cache
// key so entries roll over at midnight UTC instead of 24 hours after whenever
// they happened to be built.
func (w Window) Key() string { return w.Start.Format("2006-01-02") }

// WindowUTC returns the closed window for the given time_range, in complete
// UTC days ending at midnight today.
//
// This replaces the deprecated model.GetStartOfDayForTimeRange (deleted in
// 26617fcbdc), which evaluated the boundary in the requester's timezone. Three
// deliberate departures, all forced by serving team insights from a snapshot
// shared across the team:
//
//   - UTC rather than the caller's timezone. One cached entry has to be
//     correct for everyone on the team; a timezone-dependent window would make
//     it correct only for whoever happened to populate it.
//   - Closed at midnight today, so the window never includes a partial day and
//     a snapshot of it cannot drift.
//   - "today" is no longer accepted. A since-midnight window is open-ended by
//     definition, which is exactly what a once-daily snapshot cannot answer.
//     "1_day" replaces it and means yesterday.
func WindowUTC(timeRange string) (Window, error) {
	days := NumberOfDaysForTimeRange(timeRange)
	if days == 0 {
		return Window{}, ErrInvalidTimeRange
	}
	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return Window{
		Start: startOfToday.AddDate(0, 0, -days),
		End:   startOfToday,
	}, nil
}

// NumberOfDaysForTimeRange returns the day count implied by a time_range
// value: 7 for 7_day, 28 for 28_day, 0 for anything else (including the
// retired "today").
func NumberOfDaysForTimeRange(timeRange string) int {
	switch timeRange {
	case TimeRange1Day:
		return 1
	case TimeRange7Day:
		return 7
	case TimeRange28Day:
		return 28
	default:
		return 0
	}
}
