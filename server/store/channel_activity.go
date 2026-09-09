// Channel activity — the team-wide aggregate behind Top Channels, Top
// Inactive Channels, and the channel-governance table.
//
// Unlike the per-request queries in channel.go, this one takes no userID and
// no pagination: it returns every channel in the team so the result can be
// cached once and shared by everyone on it. Private-channel visibility is
// applied at read time by intersecting with the requester's memberships,
// which preserves the per-requester numbers the older queries produced in
// SQL (INSIGHTS_REFERENCE.md §2.2).
//
// Dropping the membership join also collapses the UNION ALL of two Posts
// aggregations described in §3.6 into a single scan.

package store

import (
	"context"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// Posts are aggregated in a MATERIALIZED CTE and only then joined to
// Channels, rather than LEFT JOINed row-by-row and grouped at the end.
//
// The shape matters more than it looks. Postgres cannot estimate selectivity
// through the four JSONB `Props ->> ...` predicates, so it estimated one row
// where there were 900,000 and picked a nested loop that materialised every
// post and rescanned it once per channel. On a 490-channel team with ~1M
// posts that query took 75 seconds — past the cache's own build timeout, so
// the page would never have loaded. Aggregating first bounds the join to one
// row per channel and makes the estimate irrelevant.
//
// MessageCount and LastRealPostAt come from the same scan, split by FILTER:
// the row set for the two-year last-post lookback is a superset of the
// window, so scanning twice would be pure waste.
//
// The CTE deliberately does not restrict to the team. Any join inside it —
// even a semi-join on the team's 500 channel ids — is planned as a nested
// loop for the same estimation reason, and measured 39s against 1.3s for the
// unrestricted version. Grouping every channel and letting the outer join
// discard the rest is the cheaper shape by a factor of thirty.
//
// The cost is real but bounded: serving one team also aggregates other teams'
// channels. A future optimisation is to build every team's rows from a single
// daily pass rather than one pass per team.
//
// The LEFT JOIN onto the CTE (rather than an inner join) is what keeps
// channels with no posts in the result — a populated channel with zero recent
// messages is exactly what the governance table exists to show.
//
// LastPostAt is computed here rather than read from the denormalized
// Channels.LastPostAt column. That column is free but wrong for this purpose:
// it advances on system messages (a join or leave) and on webhook traffic, so
// a channel nobody has spoken in for months reports activity the moment
// somebody joins it — precisely the case the governance table exists to
// surface. This derived table applies the same filters as MessageCount, so
// "last post" means the last human message.
//
// The lookback is bounded below by $4 to keep it from scanning the whole
// Posts history, and above by the window end so the value cannot drift within
// a day; channels with nothing in range report 0, which the UI renders as
// "Never".

// lastPostLookbackMillis bounds how far back the "last human message" lookup
// scans, measured from the window start. Two years is far enough that
// anything older is indistinguishable from dead for governance purposes, and
// bounding it keeps the query off the full Posts history.
const lastPostLookbackMillis int64 = 2 * 365 * 24 * 60 * 60 * 1000

const channelActivityForTeamSQL = `
WITH activity AS MATERIALIZED (
	SELECT
		Posts.ChannelId,
		count(*) FILTER (WHERE Posts.CreateAt >= $2) AS MessageCount,
		count(DISTINCT Posts.UserId) FILTER (WHERE Posts.CreateAt >= $2) AS ActivePosters,
		COALESCE(max(Posts.CreateAt) FILTER (WHERE Posts.CreateAt >= $2), 0) AS LastPostInWindow,
		max(Posts.CreateAt) AS LastRealPostAt
	FROM Posts
	WHERE Posts.DeleteAt = 0
		AND Posts.CreateAt >= $3
		AND Posts.CreateAt < $4
		AND Posts.Type = ''
		AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
		AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
		AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
		AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
	GROUP BY Posts.ChannelId
),
members AS MATERIALIZED (
	SELECT ChannelMembers.ChannelId, count(*) AS MemberCount
	FROM ChannelMembers
	JOIN Channels AS MemberChannels
		ON MemberChannels.Id = ChannelMembers.ChannelId
		AND MemberChannels.TeamId = $1
	GROUP BY ChannelMembers.ChannelId
)
SELECT
	Channels.Id,
	Channels.Type,
	Channels.DisplayName,
	Channels.Name,
	Channels.Purpose,
	Channels.Header,
	Channels.CreateAt,
	COALESCE(activity.LastRealPostAt, 0) AS LastPostAt,
	COALESCE(activity.LastPostInWindow, 0) AS LastPostInWindow,
	COALESCE(activity.MessageCount, 0) AS MessageCount,
	COALESCE(activity.ActivePosters, 0) AS ActivePosters,
	COALESCE(members.MemberCount, 0) AS MemberCount
FROM Channels
LEFT JOIN activity ON activity.ChannelId = Channels.Id
LEFT JOIN members ON members.ChannelId = Channels.Id
WHERE Channels.TeamId = $1
	AND Channels.DeleteAt = 0
	AND (Channels.Type = 'O' OR Channels.Type = 'P')
ORDER BY MessageCount DESC, Channels.Name ASC
`

// ChannelActivityForTeam returns one row per non-deleted channel in the team,
// including channels with no posts in the window — a channel with members and
// zero messages is precisely what the governance table exists to surface.
//
// The window is closed (see insights.Window), so the result describes a period
// that has already ended and is identical no matter when it is computed. That
// is what makes one snapshot valid for a whole day.
//
// Results are unpaginated by design: the caller caches the whole slice and
// filters, sorts, and pages it in memory.
func (s *Store) ChannelActivityForTeam(ctx context.Context, teamID string, w insights.Window) ([]*insights.ChannelActivity, error) {
	start, end := w.StartMillis(), w.EndMillis()
	lastPostLookback := start - lastPostLookbackMillis
	rows, err := s.replica.QueryContext(ctx, channelActivityForTeamSQL, teamID, start, lastPostLookback, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []*insights.ChannelActivity
	for rows.Next() {
		var c insights.ChannelActivity
		if err := rows.Scan(
			&c.ID, &c.Type, &c.DisplayName, &c.Name,
			&c.Purpose, &c.Header, &c.CreateAt, &c.LastPostAt, &c.LastPostInWindow,
			&c.MessageCount, &c.ActivePosters, &c.MemberCount,
		); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
