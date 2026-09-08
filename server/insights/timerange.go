package insights

import (
	"errors"
	"time"
)

// ErrInvalidTimeRange is returned when a request's time_range query param is
// not one of the supported values.
var ErrInvalidTimeRange = errors.New("time_range must be one of: 7_day, 28_day")

// StartOfWindowUTC returns the inclusive start of the window for the given
// time_range, as midnight UTC.
//
// This replaces the deprecated model.GetStartOfDayForTimeRange (deleted in
// 26617fcbdc), which evaluated the boundary in the requester's timezone. Two
// deliberate departures, both forced by serving team insights from a snapshot
// shared across the team:
//
//   - UTC rather than the caller's timezone. One cached entry has to be
//     correct for everyone on the team; a timezone-dependent window would make
//     it correct only for whoever happened to populate it.
//   - "today" is no longer accepted. A snapshot rebuilt once a day cannot
//     answer a since-midnight question — it would read near-zero just after a
//     rebuild and a full day stale just before the next one.
//
// The boundary is exclusive of today: 7_day starts at midnight seven days ago,
// so the window covers the seven complete days preceding today. Queries apply
// no upper bound, so a snapshot also picks up however much of the current day
// had elapsed when it was built, then holds that value until it is rebuilt.
func StartOfWindowUTC(timeRange string) (time.Time, error) {
	days := NumberOfDaysForTimeRange(timeRange)
	if days == 0 {
		return time.Time{}, ErrInvalidTimeRange
	}
	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return startOfToday.AddDate(0, 0, -days), nil
}

// NumberOfDaysForTimeRange returns the day count implied by a time_range
// value: 7 for 7_day, 28 for 28_day, 0 for anything else (including the
// retired "today").
func NumberOfDaysForTimeRange(timeRange string) int {
	switch timeRange {
	case TimeRange7Day:
		return 7
	case TimeRange28Day:
		return 28
	default:
		return 0
	}
}
