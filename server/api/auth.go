package api

import (
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"
)

// userIDHeader is set by the Mattermost plugin host on every authenticated
// HTTP request that reaches the plugin. Empty header means the request is
// unauthenticated.
const userIDHeader = "Mattermost-User-Id"

// authenticatedHandler is the signature handlers expect after requireUser
// extracts the caller's user ID from the request header.
type authenticatedHandler func(w http.ResponseWriter, r *http.Request, userID string)

// requireUser rejects unauthenticated requests with 401 and forwards the
// caller's user ID into the handler.
func (a *API) requireUser(h authenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get(userIDHeader)
		if userID == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		h(w, r, userID)
	}
}

// rejectGuest returns true (and writes a 403) if the caller is a guest. Guests
// were rejected by the original Insights handlers; we preserve that behavior.
func (a *API) rejectGuest(w http.ResponseWriter, userID string) bool {
	user, err := a.auth.GetUser(userID)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "user lookup failed")
		return true
	}
	if user.IsGuest() {
		writeJSONError(w, http.StatusForbidden, "guests cannot access insights")
		return true
	}
	return false
}

// requireProfessionalLicense returns true (and writes a 403) if the running
// server is not licensed at Professional or above.
func (a *API) requireProfessionalLicense(w http.ResponseWriter) bool {
	license := a.auth.GetLicense()
	if !model.MinimumProfessionalLicense(license) {
		writeJSONError(w, http.StatusForbidden, "team-scoped insights require a Mattermost Professional or Enterprise license")
		return true
	}
	return false
}

// requireTeamPermission returns true (and writes a 403) if the user does not
// hold the requested permission on the team.
func (a *API) requireTeamPermission(w http.ResponseWriter, userID, teamID string, permission *model.Permission) bool {
	if !a.auth.HasPermissionToTeam(userID, teamID, permission) {
		writeJSONError(w, http.StatusForbidden, "missing team permission")
		return true
	}
	return false
}
