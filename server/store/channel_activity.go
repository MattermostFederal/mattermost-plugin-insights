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

// The Posts predicates live in the LEFT JOIN's ON clause rather than in
// WHERE. In WHERE they would filter out the channel row itself whenever a
// channel has no qualifying posts, and a populated channel with zero recent
// messages is exactly what the governance table needs to show.
//
// MemberCount comes from a pre-aggregated derived table instead of a second
// LEFT JOIN onto ChannelMembers, which would multiply the Posts rows by the
// member count and inflate every aggregate.
//
// LastPostAt is computed here rather than read from the denormalized
// Channels.LastPostAt column. That column is free but wrong for this purpose:
// it advances on system messages (a join or leave) and on webhook traffic, so
// a channel nobody has spoken in for months reports activity the moment
// somebody joins it — precisely the case the governance table exists to
// surface. This derived table applies the same filters as MessageCount, so
// "last post" means the last human message.
//
// The lookback is bounded by $3 to keep it from scanning the whole Posts
// history; channels with nothing newer report 0, which the UI renders as
// "Never".
// lastPostLookbackMillis bounds how far back the "last human message" lookup
// scans, measured from the window start. Two years is far enough that
// anything older is indistinguishable from dead for governance purposes, and
// bounding it keeps the query off the full Posts history.
const lastPostLookbackMillis int64 = 2 * 365 * 24 * 60 * 60 * 1000

const channelActivityForTeamSQL = `
SELECT
	Channels.Id,
	Channels.Type,
	Channels.DisplayName,
	Channels.Name,
	Channels.Purpose,
	Channels.Header,
	Channels.CreateAt,
	COALESCE(LastReal.LastRealPostAt, 0) AS LastPostAt,
	COALESCE(max(Posts.CreateAt), 0) AS LastPostInWindow,
	count(Posts.Id) AS MessageCount,
	count(DISTINCT Posts.UserId) AS ActivePosters,
	COALESCE(Members.MemberCount, 0) AS MemberCount
FROM Channels
LEFT JOIN Posts
	ON Posts.ChannelId = Channels.Id
	AND Posts.DeleteAt = 0
	AND Posts.CreateAt > $2
	AND Posts.Type = ''
	AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
	AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
	AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
	AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
LEFT JOIN (
	SELECT ChannelMembers.ChannelId, count(*) AS MemberCount
	FROM ChannelMembers
	JOIN Channels AS MemberChannels
		ON MemberChannels.Id = ChannelMembers.ChannelId
		AND MemberChannels.TeamId = $1
	GROUP BY ChannelMembers.ChannelId
) AS Members ON Members.ChannelId = Channels.Id
LEFT JOIN (
	SELECT Posts.ChannelId, max(Posts.CreateAt) AS LastRealPostAt
	FROM Posts
	JOIN Channels AS PostChannels
		ON PostChannels.Id = Posts.ChannelId
		AND PostChannels.TeamId = $1
	WHERE Posts.DeleteAt = 0
		AND Posts.CreateAt > $3
		AND Posts.Type = ''
		AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
		AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
		AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
		AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
	GROUP BY Posts.ChannelId
) AS LastReal ON LastReal.ChannelId = Channels.Id
WHERE Channels.TeamId = $1
	AND Channels.DeleteAt = 0
	AND (Channels.Type = 'O' OR Channels.Type = 'P')
GROUP BY
	Channels.Id, Channels.Type, Channels.DisplayName, Channels.Name,
	Channels.Purpose, Channels.Header, Channels.CreateAt,
	LastReal.LastRealPostAt, Members.MemberCount
ORDER BY MessageCount DESC, Channels.Name ASC
`

// ChannelActivityForTeam returns one row per non-deleted channel in the team,
// including channels with no posts in the window — a channel with members and
// zero messages is precisely what the governance table exists to surface.
//
// Results are unpaginated by design: the caller caches the whole slice and
// filters, sorts, and pages it in memory.
func (s *Store) ChannelActivityForTeam(ctx context.Context, teamID string, since int64) ([]*insights.ChannelActivity, error) {
	lastPostLookback := since - lastPostLookbackMillis
	rows, err := s.replica.QueryContext(ctx, channelActivityForTeamSQL, teamID, since, lastPostLookback)
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
