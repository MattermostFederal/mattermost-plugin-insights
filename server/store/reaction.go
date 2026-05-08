package store

import (
	"context"
	"database/sql"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// SQL ported from server/channels/store/sqlstore/reaction_store.go in commit
// 26617fcbdc, with the question-mark placeholders rewritten in Postgres
// dollar form. Behavior is intended to match the original verbatim.

const topReactionsForUserNoTeamSQL = `
	SELECT EmojiName, count(EmojiName) AS Count
	FROM Reactions
	WHERE Reactions.DeleteAt = 0
		AND Reactions.UserId = $1
		AND Reactions.CreateAt > $2
	GROUP BY Reactions.EmojiName
	ORDER BY Count DESC, EmojiName ASC
	LIMIT $3 OFFSET $4
`

const topReactionsForUserWithTeamSQL = `
	SELECT EmojiName, count(EmojiName) AS Count
	FROM Reactions
	INNER JOIN Channels ON Channels.Id = Reactions.ChannelId
	WHERE Reactions.DeleteAt = 0
		AND Reactions.UserId = $1
		AND (Channels.TeamId = $2 OR Channels.Type = 'D' OR Channels.Type = 'G')
		AND Reactions.CreateAt > $3
	GROUP BY EmojiName
	ORDER BY Count DESC, EmojiName ASC
	LIMIT $4 OFFSET $5
`

// Verbatim shape (down to the unusual GROUP BY EmojiName, DeleteAt, CreateAt)
// from the deprecated reaction_store.go. The grouping preserves per-row
// DeleteAt/CreateAt so the outer WHERE can apply the time-range and soft-delete
// filters at the union-result level.
const topReactionsForTeamSQL = `
	SELECT EmojiName, sum(EmojiCount) AS Count FROM (
		(SELECT EmojiName,
				count(EmojiName) AS EmojiCount,
				Reactions.DeleteAt AS DeleteAt,
				Reactions.CreateAt AS CreateAt
		 FROM ChannelMembers
		 INNER JOIN Channels ON ChannelMembers.ChannelId = Channels.Id
		 INNER JOIN Reactions ON Channels.Id = Reactions.ChannelId
		 WHERE ChannelMembers.UserId = $1
			 AND Channels.Type = 'P'
			 AND Channels.TeamId = $2
		 GROUP BY Reactions.EmojiName, Reactions.DeleteAt, Reactions.CreateAt)
		UNION ALL
		(SELECT EmojiName,
				count(EmojiName) AS EmojiCount,
				Reactions.DeleteAt AS DeleteAt,
				Reactions.CreateAt AS CreateAt
		 FROM Reactions
		 INNER JOIN PublicChannels ON Reactions.ChannelId = PublicChannels.Id
		 WHERE PublicChannels.TeamId = $3
		 GROUP BY Reactions.EmojiName, Reactions.DeleteAt, Reactions.CreateAt)
	) AS A
	WHERE DeleteAt = 0 AND CreateAt > $4
	GROUP BY EmojiName
	ORDER BY Count DESC, EmojiName ASC
	LIMIT $5 OFFSET $6
`

// TopReactionsForUserSince returns the most-used emoji reactions created by
// the given user since the given unix-millisecond timestamp.
//
// teamID is optional. When empty the result spans every channel the user has
// reacted in. When non-empty the result is restricted to channels in that
// team plus all DM and group conversations.
func (s *Store) TopReactionsForUserSince(ctx context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopReactionList, error) {
	offset := page * perPage
	limit := perPage + 1

	var rows *sql.Rows
	var err error
	if teamID == "" {
		rows, err = s.replica.QueryContext(ctx, topReactionsForUserNoTeamSQL, userID, since, limit, offset)
	} else {
		rows, err = s.replica.QueryContext(ctx, topReactionsForUserWithTeamSQL, userID, teamID, since, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanReactionList(rows, perPage)
}

// TopReactionsForTeamSince returns the most-used emoji reactions across the
// given team since the given unix-millisecond timestamp. The result includes
// reactions in any public channel of the team plus reactions in private
// channels of the team where the calling user is a member.
func (s *Store) TopReactionsForTeamSince(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopReactionList, error) {
	offset := page * perPage
	limit := perPage + 1

	rows, err := s.replica.QueryContext(ctx, topReactionsForTeamSQL, userID, teamID, teamID, since, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanReactionList(rows, perPage)
}

func scanReactionList(rows *sql.Rows, perPage int) (*insights.TopReactionList, error) {
	items := make([]*insights.TopReaction, 0, perPage+1)
	for rows.Next() {
		var r insights.TopReaction
		if err := rows.Scan(&r.EmojiName, &r.Count); err != nil {
			return nil, err
		}
		items = append(items, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	page, hasNext := insights.Paginate(items, perPage)
	return &insights.TopReactionList{
		ListData: insights.ListData{HasNext: hasNext},
		Items:    page,
	}, nil
}
