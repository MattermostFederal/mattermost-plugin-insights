package insights

import (
	"errors"
	"time"
)

// ErrInvalidTimeRange is returned when a request's time_range query param is
// not one of the three supported values.
var ErrInvalidTimeRange = errors.New("time_range must be one of: today, 7_day, 28_day")

// StartOfDayForTimeRange returns the start-of-day instant for the given
// time_range, evaluated in the supplied location. Direct port of
// model.GetStartOfDayForTimeRange from the deleted server/public/model/insights.go.
func StartOfDayForTimeRange(timeRange string, location *time.Location) (*time.Time, error) {
	now := time.Now().In(location)
	resultTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	switch timeRange {
	case TimeRangeToday:
		// today: keep start of today as-is
	case TimeRange7Day:
		resultTime = resultTime.Add(-144 * time.Hour)
	case TimeRange28Day:
		resultTime = resultTime.Add(-648 * time.Hour)
	default:
		return nil, ErrInvalidTimeRange
	}
	return &resultTime, nil
}

// NumberOfDaysForTimeRange returns the day count implied by a time_range
// value: 1 for today, 7 for 7_day, 28 for 28_day, 0 otherwise.
func NumberOfDaysForTimeRange(timeRange string) int {
	switch timeRange {
	case TimeRangeToday:
		return 1
	case TimeRange7Day:
		return 7
	case TimeRange28Day:
		return 28
	default:
		return 0
	}
}
