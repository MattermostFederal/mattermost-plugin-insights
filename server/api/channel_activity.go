// The cached read path shared by Top Channels, Top Inactive Channels, and the
// channel-governance table.
//
// Shape of a request:
//
//	cache[channels:teamID:range:date] -> every channel in the team, unfiltered
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
	"net/url"
	"sort"
	"strings"

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
func (a *API) visibleChannelActivity(ctx context.Context, userID, teamID, timeRange string, w insights.Window) ([]*insights.ChannelActivity, error) {
	// The window's date is part of the key. Without it an entry built at
	// 23:00 would keep serving yesterday's window until 23:00 the next day;
	// with it, every range rolls over cleanly at midnight UTC and the stale
	// key falls out through the cache's sweep.
	key := cache.Key(cacheKeyChannelActivity, teamID, timeRange+":"+w.Key())
	all, err := cache.GetOrBuild(ctx, a.cache, key,
		func(buildCtx context.Context) ([]*insights.ChannelActivity, error) {
			return a.store.ChannelActivityForTeam(buildCtx, teamID, w)
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
	sortChannelActivityBy(rows, SortByPosts, ascending)
}

// Sortable columns for the governance table. Sorting happens in memory over
// the cached slice, so it costs nothing at the database — which is the reason
// the table can offer it at all.
const (
	SortByPosts       = "posts"
	SortByActiveUsers = "active_users"
	SortByMembers     = "members"
	SortByLastPost    = "last_post"
	SortByCreated     = "created"
	SortByName        = "name"
)

// channelSortLabel is the text the Channel column renders, case-folded. Sorting
// that column has to follow what is on screen: a channel renamed to "Alpha"
// keeps whatever slug it was created with, so ordering by Name would look
// arbitrary to the person reading the table.
func channelSortLabel(c *insights.ChannelActivity) string {
	if c.DisplayName != "" {
		return strings.ToLower(c.DisplayName)
	}
	return strings.ToLower(c.Name)
}

// sortChannelActivityBy orders rows by the named column. Name is the tiebreak
// for every column, so the order is total and paging is stable — without it,
// two channels with equal counts could swap between page requests and a row
// could appear twice or not at all.
func sortChannelActivityBy(rows []*insights.ChannelActivity, column string, ascending bool) {
	less := func(i, j int) (bool, bool) { // (result, decided)
		a, b := rows[i], rows[j]
		switch column {
		case SortByActiveUsers:
			return a.ActivePosters < b.ActivePosters, a.ActivePosters != b.ActivePosters
		case SortByMembers:
			return a.MemberCount < b.MemberCount, a.MemberCount != b.MemberCount
		case SortByLastPost:
			return a.LastPostAt < b.LastPostAt, a.LastPostAt != b.LastPostAt
		case SortByCreated:
			return a.CreateAt < b.CreateAt, a.CreateAt != b.CreateAt
		case SortByName:
			// Decided here rather than deferred to the tiebreak below, which
			// is always ascending — deferring left the direction toggle inert
			// on this one column.
			x, y := channelSortLabel(a), channelSortLabel(b)
			return x < y, x != y
		default: // SortByPosts
			return a.MessageCount < b.MessageCount, a.MessageCount != b.MessageCount
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		result, decided := less(i, j)
		if !decided {
			// Ties fall back to name, always ascending, so the order does not
			// flip with the direction toggle.
			return rows[i].Name < rows[j].Name
		}
		if ascending {
			return result
		}
		return !result
	})
}

// parseSort reads the sort column and direction from the query string.
// Unknown columns fall back to post count rather than erroring: a stale
// bookmark should still render a sensible table.
func parseSort(q url.Values) (column string, ascending bool) {
	switch q.Get("sort") {
	case SortByActiveUsers:
		column = SortByActiveUsers
	case SortByMembers:
		column = SortByMembers
	case SortByLastPost:
		column = SortByLastPost
	case SortByCreated:
		column = SortByCreated
	case SortByName:
		column = SortByName
	default:
		column = SortByPosts
	}
	// Descending unless asked otherwise, which is what "top" means. The
	// client always sends a direction; it is the one that decides that Channel
	// opens A→Z while the counts open at the largest.
	ascending = q.Get("direction") == "asc"
	return column, ascending
}

// Filters for the governance table's work queues. The rows are already in
// memory, so filtering costs nothing at the database — which is what makes it
// worth offering rather than making people page through hundreds of channels
// looking for the handful that need attention.
const (
	FilterUnlabelled = "unlabelled" // no purpose set
	FilterInactive   = "inactive"   // no posts in the window
	FilterPrivate    = "private"
	FilterPublic     = "public"
)

// filterChannelActivity narrows rows by a free-text search over name, display
// name and purpose, plus an optional named filter. Both are applied before
// paging, and the summary is computed before either, so the totals keep
// describing the whole team.
func filterChannelActivity(rows []*insights.ChannelActivity, search, filter string) []*insights.ChannelActivity {
	if search == "" && filter == "" {
		return rows
	}
	needle := strings.ToLower(strings.TrimSpace(search))

	out := make([]*insights.ChannelActivity, 0, len(rows))
	for _, c := range rows {
		if needle != "" {
			hay := strings.ToLower(c.Name + " " + c.DisplayName + " " + c.Purpose)
			if !strings.Contains(hay, needle) {
				continue
			}
		}
		switch filter {
		case FilterUnlabelled:
			if c.Purpose != "" {
				continue
			}
		case FilterInactive:
			if c.MessageCount > 0 {
				continue
			}
		case FilterPrivate:
			if c.Type != model.ChannelTypePrivate {
				continue
			}
		case FilterPublic:
			if c.Type != model.ChannelTypeOpen {
				continue
			}
		}
		out = append(out, c)
	}
	return out
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
