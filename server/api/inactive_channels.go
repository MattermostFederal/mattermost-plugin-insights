package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
)

// handleTopInactiveChannelsForUser handles
// GET /plugins/insights/api/v1/users/me/top/inactive_channels
func (a *API) handleTopInactiveChannelsForUser(w http.ResponseWriter, r *http.Request, userID string) {
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
	res, err := a.store.TopInactiveChannelsForUserSince(r.Context(), userID, teamID, since, params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleTopInactiveChannelsForTeam handles
// GET /plugins/insights/api/v1/teams/{team_id}/top/inactive_channels
func (a *API) handleTopInactiveChannelsForTeam(w http.ResponseWriter, r *http.Request, userID string) {
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
	res, err := a.store.TopInactiveChannelsForTeamSince(r.Context(), teamID, userID, since, params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
