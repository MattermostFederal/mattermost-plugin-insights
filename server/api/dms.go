package api

import (
	"net/http"
	"strings"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// handleTopDMsForUser handles
// GET /plugins/insights/api/v1/users/me/top/dms
//
// Hydrates each row with:
//   - SecondParticipant: the *other* user's profile (UserInformation +
//     Position), looked up via the Directory.
//   - OutgoingMessageCount: how many of the DM's posts the requester wrote,
//     via the store's OutgoingDMCounts query.
//
// The store returns each row's MessageCount as 2x the true total (DM
// channels have two members, so each post is double-counted in the join).
// We divide here, matching the original post-processing in
// server/channels/store/sqlstore/post_store.go.
func (a *API) handleTopDMsForUser(w http.ResponseWriter, r *http.Request, userID string) {
	user, ok := a.requireRegularUser(w, userID)
	if !ok {
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

	res, err := a.store.TopDMsForUserSince(r.Context(), userID, window.StartMillis(), params.page, params.perPage)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if hydrateErr := a.hydrateTopDMs(r, userID, window.StartMillis(), res); hydrateErr != nil {
		writeJSONError(w, http.StatusInternalServerError, hydrateErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (a *API) hydrateTopDMs(r *http.Request, userID string, since int64, list *insights.TopDMList) error {
	if list == nil || len(list.Items) == 0 {
		return nil
	}

	// Identify the second participant per row and divide MessageCount by 2.
	channelIDs := make([]string, 0, len(list.Items))
	secondIDs := make([]string, 0, len(list.Items))
	for _, dm := range list.Items {
		dm.MessageCount /= 2
		other := otherParticipant(dm.Participants, userID)
		secondIDs = append(secondIDs, other)
		channelIDs = append(channelIDs, dm.ChannelID)
	}

	uniqueIDs := uniqueStrings(secondIDs)
	users, err := a.directory.ListUsersByIDs(uniqueIDs)
	if err != nil {
		return err
	}
	usersByID := map[string]*struct {
		id, username, firstName, lastName, nickname, position string
		lastPicture                                           int64
	}{}
	for _, u := range users {
		usersByID[u.Id] = &struct {
			id, username, firstName, lastName, nickname, position string
			lastPicture                                           int64
		}{u.Id, u.Username, u.FirstName, u.LastName, u.Nickname, u.Position, u.LastPictureUpdate}
	}

	outgoing, err := a.store.OutgoingDMCounts(r.Context(), userID, channelIDs, since)
	if err != nil {
		return err
	}

	for i, dm := range list.Items {
		dm.OutgoingMessageCount = outgoing[dm.ChannelID]
		if u, ok := usersByID[secondIDs[i]]; ok {
			dm.SecondParticipant = &insights.DMUserInformation{
				UserInformation: insights.UserInformation{
					ID:                u.id,
					LastPictureUpdate: u.lastPicture,
					FirstName:         u.firstName,
					LastName:          u.lastName,
					NickName:          u.nickname,
					Username:          u.username,
				},
				Position: u.position,
			}
		}
	}
	return nil
}

// otherParticipant picks the participant id from a comma-joined list that
// is not the requester.
func otherParticipant(participants, requesterID string) string {
	for id := range strings.SplitSeq(participants, ",") {
		if id != "" && id != requesterID {
			return id
		}
	}
	return ""
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
