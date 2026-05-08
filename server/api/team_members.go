package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
)

// handleNewTeamMembers handles
// GET /plugins/insights/api/v1/teams/{team_id}/top/team_members
//
// Auth: authenticated user, not a guest, member of the team with view_team
// permission, server licensed at Professional or above.
//
// Privacy: FirstName / LastName are returned only when either the server's
// PrivacySettings.ShowFullName is true OR the requesting user is a system
// admin. The handler decides which behavior applies and forwards a single
// boolean to the store.
func (a *API) handleNewTeamMembers(w http.ResponseWriter, r *http.Request, userID string) {
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

	res, err := a.store.NewTeamMembersSince(r.Context(), teamID, since, params.page, params.perPage, a.effectiveShowFullName(user))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// effectiveShowFullName implements the privacy override: admins always see
// full names regardless of the privacy setting.
func (a *API) effectiveShowFullName(user *model.User) bool {
	if user.IsSystemAdmin() {
		return true
	}
	return a.directory.ShowFullName()
}
