package store

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// SQL ported from server/channels/store/sqlstore/channel_store.go's
// PostCountsByDuration in commit 26617fcbdc, Postgres-only branch.
//
// Posts are bucketed by the formatted timestamp of their CreateAt, evaluated
// in the supplied location:
//   - PostsByDay   -> "YYYY-MM-DD"
//   - PostsByHour  -> "YYYY-MM-DDTHH"  (HH24)
// The integration filter on Posts.Props matches what TopChannels uses.

// PostCountsByDuration returns post counts per (channelID, duration-bucket)
// for the given channel ids within the inclusive-start, exclusive-end window,
// optionally filtered to posts authored by `userID` (pass empty string for
// "all authors"). The grouping is "day" or "hour"; any other value yields
// the day grouping.
//
// `location` is the timezone used to bucket timestamps — typically the
// requesting user's timezone, taken from `*model.User.GetTimezoneLocation()`.
func (s *Store) PostCountsByDuration(ctx context.Context, channelIDs []string, startUnixMillis, endUnixMillis int64, userID, grouping string, location string) ([]*insights.DurationPostCount, error) {
	if len(channelIDs) == 0 {
		return nil, nil
	}
	loc := location
	if loc == "" || loc == "Local" {
		loc = "UTC"
	}
	format := "YYYY-MM-DD"
	if grouping == insights.PostsByHour {
		format = `YYYY-MM-DD"T"HH24`
	}

	durationSelect := fmt.Sprintf(`TO_CHAR(TO_TIMESTAMP(Posts.CreateAt / 1000) AT TIME ZONE '%s', '%s') AS duration`, loc, format)

	q := s.Builder.
		Select("Posts.ChannelId AS channelid", durationSelect, "count(Posts.Id) AS postcount").
		From("Posts").
		LeftJoin("Channels ON Posts.ChannelId = Channels.Id").
		Where(sq.And{
			sq.Eq{"Posts.DeleteAt": 0},
			sq.GtOrEq{"Posts.CreateAt": startUnixMillis},
			sq.Lt{"Posts.CreateAt": endUnixMillis},
			sq.Eq{"Posts.Type": ""},
			sq.Eq{"Channels.Id": channelIDs},
		}).
		Where(sq.Expr(
			`(Posts.Props ->> 'from_bot' IS NULL OR Posts.Props ->> 'from_bot' = 'false')
			 AND (Posts.Props ->> 'from_webhook' IS NULL OR Posts.Props ->> 'from_webhook' = 'false')
			 AND (Posts.Props ->> 'from_oauth_app' IS NULL OR Posts.Props ->> 'from_oauth_app' = 'false')
			 AND (Posts.Props ->> 'from_plugin' IS NULL OR Posts.Props ->> 'from_plugin' = 'false')`,
		)).
		GroupBy("channelid", "duration").
		OrderBy("channelid", "duration")

	if userID != "" {
		q = q.Where(sq.Eq{"Posts.UserId": userID})
	}

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.replica.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := []*insights.DurationPostCount{}
	for rows.Next() {
		var d insights.DurationPostCount
		if scanErr := rows.Scan(&d.ChannelID, &d.Duration, &d.PostCount); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}
