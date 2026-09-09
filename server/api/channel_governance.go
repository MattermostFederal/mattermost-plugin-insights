// Channel governance — the per-channel table behind the customer's request
// for "channel name, active users, # posts, purpose/channel metadata".
//
// It reads the same cached aggregate as Top Channels and Top Inactive
// Channels (channel_activity.go); the difference is what it shows rather than
// what it computes. Two behaviours are specific to it:
//
//   - Every channel is listed, not a top-N. A governance audit is about the
//     long tail — the channel with 80 members and no posts — so truncating
//     to the busiest few would remove the point.
//   - Labelling coverage is summarised over the whole set, not the page.
//     "38 of 412 channels have a purpose" cannot be computed client-side,
//     because the client only ever receives one page.

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// handleChannelGovernance handles
// GET /plugins/insights/api/v1/teams/{team_id}/channel_activity
//
// Auth: same gates as every other team-scoped route — authenticated
// non-guest, Professional+ license, member of the team with view_team.
func (a *API) handleChannelGovernance(w http.ResponseWriter, r *http.Request, userID string) {
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

	rows, err := a.visibleChannelActivity(r.Context(), userID, teamID, params.timeRange, window)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Summarise before paging — the totals describe the team, not the page.
	summary := summarizeGovernance(rows)

	sortChannelActivity(rows, false)
	items, hasNext := pageChannelActivity(rows, params.page, params.perPage)

	writeJSON(w, http.StatusOK, &insights.ChannelGovernanceList{
		ListData: insights.ListData{HasNext: hasNext},
		Items:    items,
		Summary:  summary,
	})
}

// summarizeGovernance counts the whole visible set. Channels with no posts in
// the window are the point of the exercise, so they are counted, not skipped.
func summarizeGovernance(rows []*insights.ChannelActivity) insights.ChannelGovernanceSummary {
	s := insights.ChannelGovernanceSummary{TotalChannels: int64(len(rows))}
	for _, c := range rows {
		if c.Purpose != "" {
			s.WithPurpose++
		}
		if c.Header != "" {
			s.WithHeader++
		}
		if c.MessageCount > 0 {
			s.ActiveChannels++
		}
	}
	return s
}
