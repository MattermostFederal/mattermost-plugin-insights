package store

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// SQL ported from server/channels/store/sqlstore/post_store.go's
// GetTopDMsForUserSince in commit 26617fcbdc, with the MySQL branches
// removed. Channel.Name encodes both DM participants as `userIdA__userIdB`,
// sorted lexicographically. We exclude:
//   - self-DMs (Name = `<id>__<id>`)
//   - DMs where either participant is in `bots`
//   - archived solo-DMs (only one participant left after agg)
// The store-returned MessageCount is double the true total (channel members
// are 2x in DMs, so each post is counted twice when joined via member);
// the handler divides by 2 in post-processing.

// TopDMsForUserSince returns the user's most active DM partners by message
// count since the given unix-millisecond timestamp.
func (s *Store) TopDMsForUserSince(ctx context.Context, userID string, since int64, page, perPage int) (*insights.TopDMList, error) {
	offset := page * perPage
	limit := perPage + 1

	// Inner: pick the DM channels the user is a member of, excluding self
	// and bot DMs.
	channelSelector := s.Builder.
		Select("Channels.Id").
		From("Channels").
		Join("ChannelMembers AS cm ON cm.ChannelId = Channels.Id").
		Where(sq.And{
			sq.Expr("Channels.Type = 'D'"),
			sq.Eq{"cm.UserId": userID},
			sq.NotEq{"Channels.Name": fmt.Sprintf("%s__%s", userID, userID)},
			sq.Expr("SPLIT_PART(Channels.Name, '__', 1) NOT IN (SELECT UserId FROM Bots)"),
			sq.Expr("SPLIT_PART(Channels.Name, '__', 2) NOT IN (SELECT UserId FROM Bots)"),
		})

	topBuilder := s.Builder.
		Select(
			"count(p.Id) AS MessageCount",
			"string_agg(distinct cm.UserId, ',') AS Participants",
			"vch.Id AS ChannelId",
		).
		FromSelect(channelSelector, "vch").
		Join("ChannelMembers AS cm ON cm.ChannelId = vch.Id").
		Join("Posts AS p ON p.ChannelId = vch.Id").
		Where(sq.And{
			sq.Gt{"p.UpdateAt": since},
			sq.Eq{"p.DeleteAt": 0},
		}).
		GroupBy("vch.Id")

	// Outer: filter archived solo-DMs (Participants must contain a comma,
	// i.e. at least 2 distinct user ids).
	outer := s.Builder.
		Select("MessageCount", "Participants", "ChannelId").
		FromSelect(topBuilder, "top_dms").
		Where(sq.Expr("POSITION(',' IN Participants) > 0")).
		OrderBy("MessageCount DESC").
		Limit(uint64(limit)).  //nolint:gosec
		Offset(uint64(offset)) //nolint:gosec

	sqlStr, args, err := outer.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.replica.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*insights.TopDM, 0, perPage+1)
	for rows.Next() {
		var dm insights.TopDM
		if scanErr := rows.Scan(&dm.MessageCount, &dm.Participants, &dm.ChannelID); scanErr != nil {
			return nil, scanErr
		}
		items = append(items, &dm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	page2, hasNext := insights.Paginate(items, perPage)
	return &insights.TopDMList{
		ListData: insights.ListData{HasNext: hasNext},
		Items:    page2,
	}, nil
}

// OutgoingDMCounts returns, for each given DM channel id, the number of
// posts the given user authored in it since the given timestamp.
func (s *Store) OutgoingDMCounts(ctx context.Context, userID string, channelIDs []string, since int64) (map[string]int64, error) {
	out := map[string]int64{}
	if len(channelIDs) == 0 {
		return out, nil
	}

	q := s.Builder.
		Select("ch.Id AS ChannelId", "count(p.Id) AS MessageCount").
		From("Channels AS ch").
		Join("Posts AS p ON p.ChannelId = ch.Id").
		Where(sq.And{
			sq.Gt{"p.UpdateAt": since},
			sq.Eq{"p.DeleteAt": 0},
			sq.Eq{"ch.Id": channelIDs},
			sq.Eq{"p.UserId": userID},
		}).
		GroupBy("ch.Id")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.replica.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var channelID string
		var count int64
		if scanErr := rows.Scan(&channelID, &count); scanErr != nil {
			return nil, scanErr
		}
		out[channelID] = count
	}
	return out, rows.Err()
}

// suppress sql import warning for now in case future helpers need it.
var _ = sql.ErrNoRows
