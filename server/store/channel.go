package store

import (
	"context"
	"database/sql"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// postgresPropsBotFilter excludes posts authored by integrations: bots,
// outgoing webhooks, OAuth apps, and plugin-issued posts. The original
// MySQL branch (JSON_EXTRACT) lives in commit 26617fcbdc; we omit it since
// supported Mattermost versions are Postgres-only.
const postgresPropsBotFilter = `
	AND (Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
	AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
	AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
	AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')
`

const topChannelsForTeamSQL = `
SELECT ID, Type, DisplayName, Name, TeamID, MessageCount FROM (
	(SELECT
		Posts.ChannelId AS ID,
		'O' AS Type,
		PublicChannels.DisplayName AS DisplayName,
		PublicChannels.Name AS Name,
		PublicChannels.TeamId AS TeamID,
		count(Posts.Id) AS MessageCount,
		0 AS DeleteAt
	FROM Posts
	LEFT JOIN PublicChannels ON Posts.ChannelId = PublicChannels.Id
	WHERE Posts.DeleteAt = 0
		AND Posts.CreateAt > $1
		AND Posts.Type = ''
		` + postgresPropsBotFilter + `
		AND PublicChannels.TeamId = $2
	GROUP BY Posts.ChannelId, PublicChannels.DisplayName, PublicChannels.Name, PublicChannels.TeamId)
	UNION ALL
	(SELECT
		Posts.ChannelId AS ID,
		Channels.Type AS Type,
		Channels.DisplayName AS DisplayName,
		Channels.Name AS Name,
		Channels.TeamId AS TeamID,
		count(Posts.Id) AS MessageCount,
		Channels.DeleteAt AS DeleteAt
	FROM Posts
	LEFT JOIN Channels ON Posts.ChannelId = Channels.Id
	LEFT JOIN ChannelMembers ON Posts.ChannelId = ChannelMembers.ChannelId
	WHERE Posts.DeleteAt = 0
		AND Posts.CreateAt > $3
		AND Posts.Type = ''
		` + postgresPropsBotFilter + `
		AND Channels.TeamId = $4
		AND Channels.Type = 'P'
		AND ChannelMembers.UserId = $5
	GROUP BY Posts.ChannelId, Channels.Type, Channels.DisplayName, Channels.Name, Channels.TeamId, Channels.DeleteAt)
) AS A
WHERE DeleteAt = 0
ORDER BY MessageCount DESC, Name ASC
LIMIT $6 OFFSET $7
`

const topChannelsForUserBaseSQL = `
SELECT
	Posts.ChannelId AS ID,
	Channels.Type AS Type,
	Channels.DisplayName AS DisplayName,
	Channels.Name AS Name,
	Channels.TeamId AS TeamID,
	count(Posts.Id) AS MessageCount
FROM Posts
LEFT JOIN Channels ON Posts.ChannelId = Channels.Id
LEFT JOIN ChannelMembers ON Posts.ChannelId = ChannelMembers.ChannelId
WHERE Posts.DeleteAt = 0
	AND Posts.CreateAt > $1
	AND Posts.Type = ''
	AND Posts.UserId = $2
	AND Channels.DeleteAt = 0
	AND (Channels.Type = 'O' OR Channels.Type = 'P')
	AND ChannelMembers.UserId = $3
` + postgresPropsBotFilter

// TopChannelsForTeamSince returns the most active channels by message count
// across the given team since the given unix-millisecond timestamp.
func (s *Store) TopChannelsForTeamSince(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopChannelList, error) {
	offset := page * perPage
	limit := perPage + 1

	rows, err := s.replica.QueryContext(ctx, topChannelsForTeamSQL,
		since, teamID,
		since, teamID, userID,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanChannelList(rows, perPage)
}

// TopChannelsForUserSince returns the most active channels by the given
// user's message count since the given unix-millisecond timestamp.
func (s *Store) TopChannelsForUserSince(ctx context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopChannelList, error) {
	offset := page * perPage
	limit := perPage + 1

	q := topChannelsForUserBaseSQL
	args := []any{since, userID, userID}
	if teamID != "" {
		q += `
		AND Channels.TeamId = $4`
		args = append(args, teamID)
	}
	q += `
GROUP BY Posts.ChannelId, Channels.Type, Channels.DisplayName, Channels.Name, Channels.TeamId
ORDER BY MessageCount DESC, Name ASC`
	if teamID != "" {
		q += `
LIMIT $5 OFFSET $6`
	} else {
		q += `
LIMIT $4 OFFSET $5`
	}
	args = append(args, limit, offset)

	rows, err := s.replica.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanChannelList(rows, perPage)
}

func scanChannelList(rows *sql.Rows, perPage int) (*insights.TopChannelList, error) {
	items := make([]*insights.TopChannel, 0, perPage+1)
	for rows.Next() {
		var c insights.TopChannel
		if err := rows.Scan(&c.ID, &c.Type, &c.DisplayName, &c.Name, &c.TeamID, &c.MessageCount); err != nil {
			return nil, err
		}
		items = append(items, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	page, hasNext := insights.Paginate(items, perPage)
	return &insights.TopChannelList{
		ListData: insights.ListData{HasNext: hasNext},
		Items:    page,
	}, nil
}
