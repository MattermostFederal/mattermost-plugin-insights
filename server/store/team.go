package store

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// SQL ported from server/channels/store/sqlstore/team_store.go in commit
// 26617fcbdc. The original was squirrel-built; we keep that shape since the
// optional FirstName/LastName columns are conditionally appended.

func newTeamMembersSelect(builder sq.StatementBuilderType, teamID string, since int64, columns ...string) sq.SelectBuilder {
	return builder.
		Select(columns...).
		From("teammembers").
		Join("users ON users.id = teammembers.userid").
		LeftJoin("bots ON bots.userid = users.id").
		Where(sq.GtOrEq{"teammembers.createat": since}).
		Where(sq.Eq{"teammembers.deleteat": 0, "teammembers.teamid": teamID, "users.deleteat": 0, "bots.userid": nil})
}

// NewTeamMembersSince returns the users who joined the given team on or
// after the given unix-millisecond timestamp, ordered most-recent-first,
// excluding bots and deleted users. The returned list's TotalCount counts
// every qualifying member, regardless of pagination.
//
// showFullName controls whether FirstName / LastName are populated; when
// false the caller should be a non-admin and these fields stay zero.
func (s *Store) NewTeamMembersSince(ctx context.Context, teamID string, since int64, page, perPage int, showFullName bool) (*insights.NewTeamMembersList, error) {
	offset := page * perPage
	limit := perPage + 1

	countSQL, countArgs, err := newTeamMembersSelect(s.Builder, teamID, since, "count(*)").ToSql()
	if err != nil {
		return nil, err
	}
	var totalCount int64
	if scanErr := s.replica.QueryRowContext(ctx, countSQL, countArgs...).Scan(&totalCount); scanErr != nil {
		return nil, scanErr
	}

	cols := []string{
		"users.id",
		"users.username",
		"users.position",
		"users.lastpictureupdate",
		"teammembers.createat",
		"users.nickname",
	}
	if showFullName {
		cols = append(cols, "users.firstname", "users.lastname")
	}
	listBuilder := newTeamMembersSelect(s.Builder, teamID, since, cols...).
		OrderBy("teammembers.createat DESC").
		Limit(uint64(limit)).  //nolint:gosec
		Offset(uint64(offset)) //nolint:gosec

	listSQL, listArgs, err := listBuilder.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.replica.QueryContext(ctx, listSQL, listArgs...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items, err := scanNewTeamMembers(rows, showFullName)
	if err != nil {
		return nil, err
	}

	page2, hasNext := insights.Paginate(items, perPage)
	return &insights.NewTeamMembersList{
		ListData:   insights.ListData{HasNext: hasNext},
		Items:      page2,
		TotalCount: totalCount,
	}, nil
}

func scanNewTeamMembers(rows *sql.Rows, showFullName bool) ([]*insights.NewTeamMember, error) {
	items := []*insights.NewTeamMember{}
	for rows.Next() {
		var ntm insights.NewTeamMember
		var dest []any
		dest = append(dest, &ntm.ID, &ntm.Username, &ntm.Position, &ntm.LastPictureUpdate, &ntm.CreateAt, &ntm.Nickname)
		if showFullName {
			dest = append(dest, &ntm.FirstName, &ntm.LastName)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		items = append(items, &ntm)
	}
	return items, rows.Err()
}
