package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
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
	rows, err := a.visibleChannelActivity(r.Context(), userID, teamID, params.timeRange, since)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// The deprecated query's `Channels.CreateAt < since` filter, preserved:
	// a channel created inside the window cannot fairly be called inactive
	// over that window. Applied here rather than in SQL because the cached
	// aggregate is shared with surfaces that want every channel.
	eligible := make([]*insights.ChannelActivity, 0, len(rows))
	for _, c := range rows {
		if c.CreateAt < since {
			eligible = append(eligible, c)
		}
	}

	sortChannelActivity(eligible, true)
	items, hasNext := pageChannelActivity(eligible, params.page, params.perPage)

	res := &insights.TopInactiveChannelList{
		ListData: insights.ListData{HasNext: hasNext},
		Items:    make([]*insights.TopInactiveChannel, 0, len(items)),
	}
	for _, c := range items {
		res.Items = append(res.Items, &insights.TopInactiveChannel{
			ID:          c.ID,
			Type:        c.Type,
			DisplayName: c.DisplayName,
			Name:        c.Name,
			// Windowed, not all-time — see insights.ChannelActivity.
			LastActivityAt: c.LastPostInWindow,
			MessageCount:   c.MessageCount,
			Participants:   []string{},
		})
	}

	if err := a.store.AttachInactiveChannelParticipants(r.Context(), res.Items); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
