package api

import (
	"net/http"
	"strconv"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

const (
	defaultPerPage = 60
	maxPerPage     = 100
)

type topParams struct {
	timeRange string
	page      int
	perPage   int
}

// parseTopParams reads the time_range / page / per_page query parameters
// shared by every top-* endpoint. It writes a 400 response and returns ok=false
// on any parse / range error.
func parseTopParams(w http.ResponseWriter, r *http.Request) (topParams, bool) {
	q := r.URL.Query()

	timeRange := q.Get("time_range")
	if timeRange == "" {
		writeJSONError(w, http.StatusBadRequest, "time_range query parameter is required")
		return topParams{}, false
	}
	switch timeRange {
	case insights.TimeRangeToday, insights.TimeRange7Day, insights.TimeRange28Day:
		// ok
	default:
		writeJSONError(w, http.StatusBadRequest, "time_range must be one of: today, 7_day, 28_day")
		return topParams{}, false
	}

	page, err := parseNonNegativeInt(q.Get("page"), 0)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "page must be a non-negative integer")
		return topParams{}, false
	}

	perPage, err := parseNonNegativeInt(q.Get("per_page"), defaultPerPage)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "per_page must be a non-negative integer")
		return topParams{}, false
	}
	if perPage == 0 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	return topParams{timeRange: timeRange, page: page, perPage: perPage}, true
}

func parseNonNegativeInt(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, strconv.ErrRange
	}
	return n, nil
}

// requireRegularUser fetches the user and rejects guests. On failure it has
// already written the response and returns ok=false.
func (a *API) requireRegularUser(w http.ResponseWriter, userID string) (*model.User, bool) {
	user, err := a.auth.GetUser(userID)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "user lookup failed")
		return nil, false
	}
	if user.IsGuest() {
		writeJSONError(w, http.StatusForbidden, "guests cannot access insights")
		return nil, false
	}
	return user, true
}

// computeSinceMillis resolves time_range to a start unix-millisecond timestamp
// in the user's local timezone (falling back to UTC).
func computeSinceMillis(w http.ResponseWriter, timeRange string, user *model.User) (int64, bool) {
	loc := user.GetTimezoneLocation()
	start, err := insights.StartOfDayForTimeRange(timeRange, loc)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return 0, false
	}
	return start.UnixMilli(), true
}
