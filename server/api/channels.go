package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// handleTopChannelsForUser handles
// GET /plugins/insights/api/v1/users/me/top/channels
func (a *API) handleTopChannelsForUser(w http.ResponseWriter, r *http.Request, userID string) {
	user, ok := a.requireRegularUser(w, userID)
	if !ok {
		return
	}
	params, ok := parseTopParams(w, r)
	if !ok {
		return
	}
	teamID := r.URL.Query().Get("team_id")
	since, ok := computeSinceMillis(w, params.timeRange, user)
	if !ok {
		return
	}
	res, err := a.store.TopChannelsForUserSince(r.Context(), userID, teamID, since, params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if hydrateErr := a.attachChartData(r.Context(), res, since, params.timeRange, userID, user); hydrateErr != nil {
		writeJSONError(w, http.StatusInternalServerError, hydrateErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleTopChannelsForTeam handles
// GET /plugins/insights/api/v1/teams/{team_id}/top/channels
func (a *API) handleTopChannelsForTeam(w http.ResponseWriter, r *http.Request, userID string) {
	user, ok := a.requireRegularUser(w, userID)
	if !ok {
		return
	}
	if a.requireProfessionalLicense(w) {
		return
	}
	teamID := mux.Vars(r)["team_id"]
	if a.requireTeamPermission(w, userID, teamID, model.PermissionViewTeam) {
		return
	}
	params, ok := parseTopParams(w, r)
	if !ok {
		return
	}
	since, ok := computeSinceMillis(w, params.timeRange, user)
	if !ok {
		return
	}
	res, err := a.store.TopChannelsForTeamSince(r.Context(), teamID, userID, since, params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Team-scoped chart aggregates over all authors (no user filter).
	if hydrateErr := a.attachChartData(r.Context(), res, since, params.timeRange, "", user); hydrateErr != nil {
		writeJSONError(w, http.StatusInternalServerError, hydrateErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// attachChartData runs PostCountsByDuration over the result's channel ids
// and assembles the view-model map onto res.PostCountByDuration. If the
// result has no items, the map is initialized to {} (so JSON clients always
// see an object, never null).
func (a *API) attachChartData(ctx context.Context, res *insights.TopChannelList, sinceMillis int64, timeRange, postAuthorUserID string, user *model.User) error {
	if res == nil {
		return nil
	}
	if len(res.Items) == 0 {
		res.PostCountByDuration = insights.ChannelPostCountByDuration{}
		return nil
	}

	// Always day-grouped now. Hour grouping existed only for the "today"
	// range, which the daily snapshot retired (insights.StartOfWindowUTC).
	// Bucketing is UTC for the same reason the window is: the result is
	// shared across the team, so it cannot follow the caller's clock.
	rows, err := a.store.PostCountsByDuration(ctx, res.ChannelIDs(), sinceMillis, postAuthorUserID, insights.PostsByDay, time.UTC.String())
	if err != nil {
		return err
	}
	start := time.UnixMilli(sinceMillis).UTC()
	res.PostCountByDuration = insights.ToChannelPostCountByDuration(rows, &start, insights.NumberOfDaysForTimeRange(timeRange), res.ChannelIDs())
	return nil
}
