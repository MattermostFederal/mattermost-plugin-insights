// Top Playbooks handlers — analogous to Top Boards (boards.go).
//
// Why these live in this plugin and not the Playbooks plugin: the
// Playbooks plugin's `licenseAndGuestCheck` only accepts `professional`
// or `enterprise` SKUs, so on `advanced` (Enterprise Advanced) licenses
// its team-scope endpoint always returns 500. This plugin's
// `requireProfessionalLicense` uses `model.MinimumProfessionalLicense`
// which correctly recognizes `advanced` as a higher tier.

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// unavailablePlaybookList mirrors unavailableBoardList (boards.go) for the
// Top Playbooks routes.
func unavailablePlaybookList() *insights.TopPlaybookList {
	return &insights.TopPlaybookList{
		ListData: insights.ListData{NotAvailable: true},
		Items:    []*insights.TopPlaybook{},
	}
}

// handleTopPlaybooksForTeam handles
// GET /plugins/insights/api/v1/teams/{team_id}/top/playbooks
//
// Auth: authenticated user, not a guest, member of the team with view_team
// permission, server licensed at Professional or above.
func (a *API) handleTopPlaybooksForTeam(w http.ResponseWriter, r *http.Request, userID string) {
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

	if !EnableBoardsAndPlaybooks {
		writeJSON(w, http.StatusOK, unavailablePlaybookList())
		return
	}

	res, err := a.store.TopPlaybooksForTeam(r.Context(), teamID, userID, since, params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleTopPlaybooksForUser handles
// GET /plugins/insights/api/v1/users/me/top/playbooks
//
// Query params: time_range (required), team_id (required — Top Playbooks
// is always team-scoped, the user-variant just filters to playbooks the
// user is a member of).
//
// Auth: authenticated user, not a guest. No license requirement.
func (a *API) handleTopPlaybooksForUser(w http.ResponseWriter, r *http.Request, userID string) {
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

	if !EnableBoardsAndPlaybooks {
		writeJSON(w, http.StatusOK, unavailablePlaybookList())
		return
	}

	res, err := a.store.TopPlaybooksForUser(r.Context(), teamID, userID, since, params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
