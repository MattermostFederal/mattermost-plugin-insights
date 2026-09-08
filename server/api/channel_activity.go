// The cached read path shared by Top Channels, Top Inactive Channels, and the
// channel-governance table.
//
// Shape of a request:
//
//	cache[channels:teamID:timeRange]  -> every channel in the team, unfiltered
//	  miss -> store.ChannelActivityForTeam            (once per team per day)
//	filter to the requester's visible channels        (small indexed query)
//	sort, then paginate                               (in memory)
//
// The expensive aggregation is keyed by team, not by user, so it runs once a
// day however many people open the page. What stays per-request is the
// membership lookup and the slicing — both cheap, and both necessary, since
// two members of the same team must not see each other's private channels.

package api

import (
	"context"
	"sort"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/cache"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// cacheKeyChannelActivity namespaces the team-wide channel aggregate.
const cacheKeyChannelActivity = "channel_activity"

// visibleChannelActivity returns the team's channel aggregate for the window,
// reduced to the channels this user is allowed to see.
//
// Public channels are visible to every team member; private ones only to
// their members. The cached slice is never mutated — filtering copies into a
// fresh slice, because the same backing array is handed to every concurrent
// reader on the team.
func (a *API) visibleChannelActivity(ctx context.Context, userID, teamID, timeRange string, since int64) ([]*insights.ChannelActivity, error) {
	all, err := cache.GetOrBuild(ctx, a.cache, cache.Key(cacheKeyChannelActivity, teamID, timeRange),
		func(buildCtx context.Context) ([]*insights.ChannelActivity, error) {
			return a.store.ChannelActivityForTeam(buildCtx, teamID, since)
		})
	if err != nil {
		return nil, err
	}

	privateIDs, err := a.store.PrivateChannelIDsForUser(ctx, userID, teamID)
	if err != nil {
		return nil, err
	}
	visiblePrivate := make(map[string]struct{}, len(privateIDs))
	for _, id := range privateIDs {
		visiblePrivate[id] = struct{}{}
	}

	out := make([]*insights.ChannelActivity, 0, len(all))
	for _, c := range all {
		if c.Type == model.ChannelTypeOpen {
			out = append(out, c)
			continue
		}
		if _, ok := visiblePrivate[c.ID]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

// sortChannelActivity orders rows by message count, ascending when `ascending`
// is set (Top Inactive Channels) and descending otherwise (Top Channels).
// Name breaks ties in both directions, matching the deprecated queries'
// `ORDER BY MessageCount <dir>, Name ASC`.
func sortChannelActivity(rows []*insights.ChannelActivity, ascending bool) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].MessageCount != rows[j].MessageCount {
			if ascending {
				return rows[i].MessageCount < rows[j].MessageCount
			}
			return rows[i].MessageCount > rows[j].MessageCount
		}
		return rows[i].Name < rows[j].Name
	})
}

// pageChannelActivity applies the limit+1 convention used everywhere else in
// this plugin: take one more row than asked for, and report has_next from
// whether it existed.
func pageChannelActivity(rows []*insights.ChannelActivity, page, perPage int) (items []*insights.ChannelActivity, hasNext bool) {
	offset := page * perPage
	if offset >= len(rows) {
		return []*insights.ChannelActivity{}, false
	}
	end := offset + perPage
	if end > len(rows) {
		return rows[offset:], false
	}
	return rows[offset:end], end < len(rows)
}
