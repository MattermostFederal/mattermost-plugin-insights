package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
)

// handleTopReactionsForUser handles
// GET /plugins/insights/api/v1/users/me/top/reactions
//
// Query params: time_range (today|7_day|28_day, required), team_id (optional),
// page (default 0), per_page (default 60, capped at 100).
//
// Auth: requires an authenticated user; rejects guests.
func (a *API) handleTopReactionsForUser(w http.ResponseWriter, r *http.Request, userID string) {
	user, ok := a.requireRegularUser(w, userID)
	if !ok {
		return
	}

	params, ok := parseTopParams(w, r)
	if !ok {
		return
	}
	teamID := r.URL.Query().Get("team_id")

	window, ok := computeWindow(w, params.timeRange, user)
	if !ok {
		return
	}

	res, err := a.store.TopReactionsForUserSince(r.Context(), userID, teamID, window.StartMillis(), params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleTopReactionsForTeam handles
// GET /plugins/insights/api/v1/teams/{team_id}/top/reactions
//
// Auth: authenticated user, not a guest, member of the team with view_team
// permission, and the server is licensed at Professional or above.
func (a *API) handleTopReactionsForTeam(w http.ResponseWriter, r *http.Request, userID string) {
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
	window, ok := computeWindow(w, params.timeRange, user)
	if !ok {
		return
	}

	res, err := a.store.TopReactionsForTeamSince(r.Context(), teamID, userID, window.StartMillis(), params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
