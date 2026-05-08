package store

import (
	"context"
	"database/sql"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// SQL ported from server/channels/store/sqlstore/thread_store.go in commit
// 26617fcbdc, with question-mark placeholders rewritten as Postgres dollar
// placeholders. The grouping shape is preserved verbatim — t.ReplyCount is
// not in GROUP BY but Postgres allows it because t.PostId is the primary key
// of threads.

const topThreadsForTeamSQL = `
SELECT
	threads_list.PostId,
	threads_list.ReplyCount,
	threads_list.ChannelId,
	threads_list.DisplayName,
	threads_list.Name,
	threads_list.Participants,
	p.UserId
FROM ((
	SELECT
		t.PostId, t.ReplyCount, t.ChannelId, t.Participants,
		c.DisplayName, c.Name
	FROM Threads t
	LEFT JOIN PublicChannels c ON t.ChannelId = c.Id
	WHERE t.threaddeleteat IS NULL
		AND t.LastReplyAt > $1
		AND c.TeamId = $2
	GROUP BY t.PostId, c.DisplayName, c.Name, t.Participants
)
UNION ALL (
	SELECT
		t.PostId, t.ReplyCount, t.ChannelId, t.Participants,
		c.DisplayName, c.Name
	FROM Threads t
	LEFT JOIN ChannelMembers cm ON t.ChannelId = cm.ChannelId
	LEFT JOIN Channels c ON t.ChannelId = c.Id
	WHERE t.threaddeleteat IS NULL
		AND cm.UserId = $3
		AND c.Type = 'P'
		AND c.TeamId = $4
		AND t.LastReplyAt > $5
	GROUP BY t.PostId, c.DisplayName, c.Name, t.Participants
)) AS threads_list
LEFT JOIN Posts AS p ON p.Id = threads_list.PostId
ORDER BY ReplyCount DESC
LIMIT $6 OFFSET $7
`

const topThreadsForUserSQL = `
SELECT
	threads_list.PostId,
	threads_list.ReplyCount,
	threads_list.ChannelId,
	threads_list.DisplayName,
	threads_list.Name,
	threads_list.Participants,
	p.UserId
FROM ((
	SELECT
		t.PostId, t.ReplyCount, t.ChannelId, t.Participants,
		c.DisplayName, c.Name
	FROM Threads t
	LEFT JOIN PublicChannels c ON t.ChannelId = c.Id
	LEFT JOIN ThreadMemberships tm ON t.PostId = tm.PostId
	WHERE t.threaddeleteat IS NULL
		AND t.LastReplyAt > $1
		AND c.TeamId = $2
		AND tm.UserId = $3
		AND tm.Following = TRUE
	GROUP BY t.PostId, c.DisplayName, c.Name, t.Participants
)
UNION ALL (
	SELECT
		t.PostId, t.ReplyCount, t.ChannelId, t.Participants,
		c.DisplayName, c.Name
	FROM Threads t
	LEFT JOIN ChannelMembers cm ON t.ChannelId = cm.ChannelId
	LEFT JOIN Channels c ON t.ChannelId = c.Id
	LEFT JOIN ThreadMemberships tm ON t.PostId = tm.PostId
	WHERE cm.UserId = $4
		AND c.Type = 'P'
		AND c.TeamId = $5
		AND t.threaddeleteat IS NULL
		AND t.LastReplyAt > $6
		AND tm.UserId = $7
		AND tm.Following = TRUE
	GROUP BY t.PostId, c.DisplayName, c.Name, t.Participants
)) AS threads_list
LEFT JOIN Posts AS p ON p.Id = threads_list.PostId
ORDER BY ReplyCount DESC
LIMIT $8 OFFSET $9
`

// TopThreadsForUserSince returns the most active threads (by reply count)
// that the given user is following, since the given unix-millisecond
// timestamp. The result spans threads in:
//   - public channels of the team (where the user follows the thread), and
//   - private channels of the team where the user is a member (and follows
//     the thread).
//
// Returned items carry the bare row data (PostID, ReplyCount, ChannelID,
// channel display name + name, participants, root-post UserID). Hydrating
// each item with full UserInformation and Post is the handler's job; see
// server/api/threads.go.
func (s *Store) TopThreadsForUserSince(ctx context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopThreadList, error) {
	offset := page * perPage
	limit := perPage + 1

	rows, err := s.replica.QueryContext(ctx, topThreadsForUserSQL,
		since, teamID, userID,
		userID, teamID, since, userID,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanThreadList(rows, perPage)
}

// TopThreadsForTeamSince returns the most active threads across the given
// team since the given unix-millisecond timestamp:
//   - threads in any public channel of the team, regardless of membership;
//   - threads in private channels of the team where the calling user is a
//     member.
//
// Hydration with UserInformation and Post is the handler's responsibility.
func (s *Store) TopThreadsForTeamSince(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopThreadList, error) {
	offset := page * perPage
	limit := perPage + 1

	rows, err := s.replica.QueryContext(ctx, topThreadsForTeamSQL,
		since, teamID,
		userID, teamID, since,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanThreadList(rows, perPage)
}

func scanThreadList(rows *sql.Rows, perPage int) (*insights.TopThreadList, error) {
	items := make([]*insights.TopThread, 0, perPage+1)
	for rows.Next() {
		var t insights.TopThread
		var participants model.StringArray
		if err := rows.Scan(&t.PostID, &t.ReplyCount, &t.ChannelID, &t.DisplayName, &t.Name, &participants, &t.UserID); err != nil {
			return nil, err
		}
		t.Participants = participants
		items = append(items, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	page, hasNext := insights.Paginate(items, perPage)
	return &insights.TopThreadList{
		ListData: insights.ListData{HasNext: hasNext},
		Items:    page,
	}, nil
}
