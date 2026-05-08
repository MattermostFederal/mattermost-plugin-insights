// Ported from mattermost-plugin-boards/server/api/insights.go at commit
// c8e729b6^ (the commit that deleted board insights). The deprecated
// handler computed the same `boardIDs` ACL via `getUserBoards` and called
// `app.GetTeamBoardsInsights` / `GetUserBoardsInsights` against it; here
// we compute the ACL via the plugin's local store and dispatch directly.
//
// The Top Boards routes live on the insights plugin (not the Boards
// plugin) because the Boards plugin's HTTP routes were removed in commit
// c8e729b6 (June 2024) — this plugin queries the focalboard tables on
// the same Mattermost database directly.

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
)

// handleTopBoardsForTeam handles
// GET /plugins/insights/api/v1/teams/{team_id}/top/boards
//
// Auth: authenticated user, not a guest, member of the team with view_team
// permission, server licensed at Professional or above.
func (a *API) handleTopBoardsForTeam(w http.ResponseWriter, r *http.Request, userID string) {
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

	boardIDs, err := a.store.BoardIDsForUserInTeam(r.Context(), userID, teamID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	res, err := a.store.TopBoardsForTeam(r.Context(), teamID, boardIDs, since, params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleTopBoardsForUser handles
// GET /plugins/insights/api/v1/users/me/top/boards
//
// Query params: time_range (required), team_id (required — Top Boards is
// always team-scoped, but the user-scope variant filters to boards the
// user has touched).
//
// Auth: authenticated user, not a guest. No license requirement.
func (a *API) handleTopBoardsForUser(w http.ResponseWriter, r *http.Request, userID string) {
	user, ok := a.requireRegularUser(w, userID)
	if !ok {
		return
	}

	teamID := r.URL.Query().Get("team_id")
	if teamID == "" {
		writeJSONError(w, http.StatusBadRequest, "team_id query parameter is required")
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

	boardIDs, err := a.store.BoardIDsForUserInTeam(r.Context(), userID, teamID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	res, err := a.store.TopBoardsForUser(r.Context(), teamID, userID, boardIDs, since, params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
