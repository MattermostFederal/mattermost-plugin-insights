package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// handleTopThreadsForUser handles
// GET /plugins/insights/api/v1/users/me/top/threads
func (a *API) handleTopThreadsForUser(w http.ResponseWriter, r *http.Request, userID string) {
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

	res, err := a.store.TopThreadsForUserSince(r.Context(), userID, teamID, window.StartMillis(), params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if hydrateErr := a.hydrateThreads(r.Context(), res); hydrateErr != nil {
		writeJSONError(w, http.StatusInternalServerError, hydrateErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleTopThreadsForTeam handles
// GET /plugins/insights/api/v1/teams/{team_id}/top/threads
func (a *API) handleTopThreadsForTeam(w http.ResponseWriter, r *http.Request, userID string) {
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

	res, err := a.store.TopThreadsForTeamSince(r.Context(), teamID, userID, window.StartMillis(), params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if hydrateErr := a.hydrateThreads(r.Context(), res); hydrateErr != nil {
		writeJSONError(w, http.StatusInternalServerError, hydrateErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// hydrateThreads attaches UserInformation and Post to each item in the list.
// Mirrors the postProcessTopThreads helper from the deprecated
// server/channels/store/sqlstore/thread_store.go.
func (a *API) hydrateThreads(_ context.Context, list *insights.TopThreadList) error {
	if list == nil || len(list.Items) == 0 {
		return nil
	}

	userIDs := uniqueUserIDs(list.Items)
	users, err := a.directory.ListUsersByIDs(userIDs)
	if err != nil {
		return err
	}
	usersByID := make(map[string]*model.User, len(users))
	for _, u := range users {
		usersByID[u.Id] = u
	}

	for _, item := range list.Items {
		if u, ok := usersByID[item.UserID]; ok {
			item.UserInformation = &insights.UserInformation{
				ID:                u.Id,
				LastPictureUpdate: u.LastPictureUpdate,
				FirstName:         u.FirstName,
				LastName:          u.LastName,
				NickName:          u.Nickname,
				Username:          u.Username,
			}
		}
		post, postErr := a.directory.GetPost(item.PostID)
		if postErr != nil {
			return postErr
		}
		item.Post = post
	}
	return nil
}

func uniqueUserIDs(items []*insights.TopThread) []string {
	seen := make(map[string]struct{}, len(items))
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if _, dup := seen[item.UserID]; dup {
			continue
		}
		seen[item.UserID] = struct{}{}
		ids = append(ids, item.UserID)
	}
	return ids
}
