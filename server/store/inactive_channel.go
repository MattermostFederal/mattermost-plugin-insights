package store

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// SQL ported from server/channels/store/sqlstore/channel_store.go in commit
// 26617fcbdc. The participants post-processing query uses Postgres'
// string_agg; the MySQL GROUP_CONCAT branch is dropped.

const topInactiveChannelsForTeamSQL = `
SELECT ID, Type, DisplayName, Name, MessageCount, LastActivityAt FROM (
	(SELECT
		PublicChannels.Id AS ID,
		'O' AS Type,
		PublicChannels.DisplayName AS DisplayName,
		PublicChannels.Name AS Name,
		COALESCE(count(Posts.Id), 0) AS MessageCount,
		COALESCE(max(Posts.CreateAt), 0) AS LastActivityAt
	FROM PublicChannels
	LEFT JOIN Posts ON Posts.ChannelId = PublicChannels.Id
		AND Posts.Type = ''
		AND Posts.CreateAt > $1
		AND Posts.DeleteAt = 0
	LEFT JOIN Channels ON Channels.Id = PublicChannels.Id
	WHERE PublicChannels.TeamId = $2
		AND PublicChannels.DeleteAt = 0
		AND Channels.CreateAt < $3
	GROUP BY PublicChannels.Id, PublicChannels.DisplayName, PublicChannels.Name, PublicChannels.TeamId)
	UNION ALL
	(SELECT
		Channels.Id AS ID,
		Channels.Type AS Type,
		Channels.DisplayName AS DisplayName,
		Channels.Name AS Name,
		COALESCE(count(Posts.Id), 0) AS MessageCount,
		COALESCE(max(Posts.CreateAt), 0) AS LastActivityAt
	FROM Channels
	LEFT JOIN Posts ON Posts.ChannelId = Channels.Id
		AND Posts.Type = ''
		AND Posts.CreateAt > $4
		AND Posts.DeleteAt = 0
	LEFT JOIN ChannelMembers ON Channels.Id = ChannelMembers.ChannelId
	WHERE Channels.TeamId = $5
		AND Channels.CreateAt < $6
		AND Channels.Type = 'P'
		AND Channels.DeleteAt = 0
		AND ChannelMembers.UserId = $7
	GROUP BY Channels.Id, Channels.Type, Channels.DisplayName, Channels.Name)
) AS A
ORDER BY MessageCount ASC, Name ASC
LIMIT $8 OFFSET $9
`

const topInactiveChannelsForUserBaseSQL = `
SELECT
	Channels.Id AS ID,
	Channels.Type AS Type,
	Channels.DisplayName AS DisplayName,
	Channels.Name AS Name,
	COALESCE(count(Posts.Id), 0) AS MessageCount,
	COALESCE(max(Posts.CreateAt), 0) AS LastActivityAt
FROM Channels
LEFT JOIN Posts ON Posts.ChannelId = Channels.Id
	AND Posts.Type = ''
	AND Posts.CreateAt > $1
	AND Posts.DeleteAt = 0
LEFT JOIN ChannelMembers ON Channels.Id = ChannelMembers.ChannelId
WHERE Channels.DeleteAt = 0
	AND Channels.CreateAt < $2
	AND (Channels.Type = 'O' OR Channels.Type = 'P')
	AND ChannelMembers.UserId = $3
`

// TopInactiveChannelsForTeamSince returns channels in the team ordered by
// message count ascending — least active first.
func (s *Store) TopInactiveChannelsForTeamSince(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopInactiveChannelList, error) {
	offset := page * perPage
	limit := perPage + 1

	rows, err := s.replica.QueryContext(ctx, topInactiveChannelsForTeamSQL,
		since, teamID, since,
		since, teamID, since, userID,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return s.scanInactiveChannelList(ctx, rows, perPage)
}

// TopInactiveChannelsForUserSince returns channels the user is a member of,
// ordered by message count ascending. teamID is optional.
func (s *Store) TopInactiveChannelsForUserSince(ctx context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopInactiveChannelList, error) {
	offset := page * perPage
	limit := perPage + 1

	q := topInactiveChannelsForUserBaseSQL
	args := []any{since, since, userID}
	if teamID != "" {
		q += `
		AND Channels.TeamId = $4`
		args = append(args, teamID)
	}
	q += `
GROUP BY Channels.Id, Channels.Type, Channels.DisplayName, Channels.Name
ORDER BY MessageCount ASC, Name ASC`
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

	return s.scanInactiveChannelList(ctx, rows, perPage)
}

func (s *Store) scanInactiveChannelList(ctx context.Context, rows *sql.Rows, perPage int) (*insights.TopInactiveChannelList, error) {
	items := make([]*insights.TopInactiveChannel, 0, perPage+1)
	for rows.Next() {
		var c insights.TopInactiveChannel
		if err := rows.Scan(&c.ID, &c.Type, &c.DisplayName, &c.Name, &c.MessageCount, &c.LastActivityAt); err != nil {
			return nil, err
		}
		items = append(items, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	page, hasNext := insights.Paginate(items, perPage)
	if err := s.attachInactiveChannelParticipants(ctx, page); err != nil {
		return nil, err
	}
	return &insights.TopInactiveChannelList{
		ListData: insights.ListData{HasNext: hasNext},
		Items:    page,
	}, nil
}

// attachInactiveChannelParticipants runs the secondary string_agg query
// against ChannelMembers and populates each channel's Participants field.
// Empty string_agg results yield an empty slice, not [""].
func (s *Store) attachInactiveChannelParticipants(ctx context.Context, channels []*insights.TopInactiveChannel) error {
	if len(channels) == 0 {
		return nil
	}
	ids := make([]string, 0, len(channels))
	for _, c := range channels {
		ids = append(ids, c.ID)
	}

	q := s.Builder.
		Select("ChannelId", "string_agg(UserId, ',') AS UserIds").
		From("ChannelMembers").
		Where(sq.Eq{"ChannelId": ids}).
		GroupBy("ChannelId")
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return err
	}

	rows, err := s.replica.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	byChannel := make(map[string][]string, len(channels))
	for rows.Next() {
		var channelID string
		var userIDsAgg sql.NullString
		if scanErr := rows.Scan(&channelID, &userIDsAgg); scanErr != nil {
			return scanErr
		}
		if userIDsAgg.Valid && userIDsAgg.String != "" {
			byChannel[channelID] = splitCSV(userIDsAgg.String)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, c := range channels {
		if parts, ok := byChannel[c.ID]; ok {
			c.Participants = parts
		} else {
			c.Participants = []string{}
		}
	}
	return nil
}

func splitCSV(s string) []string {
	if s == "" {
		return []string{}
	}
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

// silence unused-import for lib/pq while we keep the option open of using
// pq.Array elsewhere; it's a transitive dep already.
var _ = pq.Array
