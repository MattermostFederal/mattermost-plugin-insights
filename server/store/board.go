package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
	"github.com/lib/pq"
)

// SQL ported from
// mattermost-plugin-boards/server/services/store/sqlstore/board_insights.go
// at commit c8e729b6^ (the commit that deleted board insights). The
// MySQL/SQLite branches in the original were stripped — this plugin is
// PostgreSQL-only.
//
// The queries union together rows from `focalboard_boards_history`
// (board-edit events) and `focalboard_blocks_history` (block-edit events,
// e.g. cards), aggregate the count per board, and return the top N by
// activity_count. The deprecated source filtered out `modified_by =
// 'system'` rows (auto-system edits) and `delete_at != 0` boards; both
// are preserved verbatim.
//
// The teamID + boardIDs filter mirrors the original: the caller (the
// API handler) computes the set of boards the user can access and passes
// the IDs in. For team-scope this is every board the user can read in
// the team; for user-scope this is the user-only filter on top of that.

const topBoardsCountSelector = `
	id, title, icon, activity_count, active_users, created_by
`

// teamBoardsSQL returns top boards for an entire team, restricted to a
// pre-computed boardIDs ACL list.
const teamBoardsSQLTemplate = `
	SELECT id, title, icon, sum(count) AS activity_count,
		   string_agg(DISTINCT modified_by, ',') AS active_users, created_by
	FROM (
		SELECT boards.id, boards.icon, boards.title,
			   count(boards_history.id) AS count,
			   boards_history.modified_by, boards.created_by
		FROM focalboard_boards_history AS boards_history
		JOIN focalboard_boards AS boards ON boards_history.id = boards.id
		WHERE boards_history.insert_at > $1
		  AND boards.team_id = $2
		  AND boards.id = ANY($3)
		  AND boards_history.modified_by != 'system'
		  AND boards.delete_at = 0
		GROUP BY boards.id, boards_history.id, boards_history.modified_by

		UNION ALL

		SELECT boards.id, boards.icon, boards.title,
			   count(blocks_history.id) AS count,
			   blocks_history.modified_by, boards.created_by
		FROM focalboard_blocks_history AS blocks_history
		JOIN focalboard_boards AS boards ON blocks_history.board_id = boards.id
		WHERE blocks_history.insert_at > $1
		  AND boards.team_id = $2
		  AND boards.id = ANY($3)
		  AND blocks_history.modified_by != 'system'
		  AND boards.delete_at = 0
		GROUP BY boards.id, blocks_history.board_id, blocks_history.modified_by
	) AS boards_and_blocks_history
	GROUP BY id, title, icon, created_by
	ORDER BY activity_count DESC
	LIMIT $4 OFFSET $5
`

// userBoardsSQL is the team query plus an outer filter restricting the
// result to boards the requesting user has either created or appears in
// the active_users (i.e. they edited the board or one of its blocks).
const userBoardsSQLTemplate = `
	SELECT %s FROM (
		SELECT id, title, icon, sum(count) AS activity_count,
			   string_agg(DISTINCT modified_by, ',') AS active_users, created_by
		FROM (
			SELECT boards.id, boards.icon, boards.title,
				   count(boards_history.id) AS count,
				   boards_history.modified_by, boards.created_by
			FROM focalboard_boards_history AS boards_history
			JOIN focalboard_boards AS boards ON boards_history.id = boards.id
			WHERE boards_history.insert_at > $1
			  AND boards.team_id = $2
			  AND boards.id = ANY($3)
			  AND boards_history.modified_by != 'system'
			  AND boards.delete_at = 0
			GROUP BY boards.id, boards_history.id, boards_history.modified_by

			UNION ALL

			SELECT boards.id, boards.icon, boards.title,
				   count(blocks_history.id) AS count,
				   blocks_history.modified_by, boards.created_by
			FROM focalboard_blocks_history AS blocks_history
			JOIN focalboard_boards AS boards ON blocks_history.board_id = boards.id
			WHERE blocks_history.insert_at > $1
			  AND boards.team_id = $2
			  AND boards.id = ANY($3)
			  AND blocks_history.modified_by != 'system'
			  AND boards.delete_at = 0
			GROUP BY boards.id, blocks_history.board_id, blocks_history.modified_by
		) AS boards_and_blocks_history
		GROUP BY id, title, icon, created_by
		ORDER BY activity_count DESC
	) AS boards_and_blocks_history_for_user
	WHERE created_by = $4 OR position($4 IN active_users) > 0
	LIMIT $5 OFFSET $6
`

// TopBoardsForTeam returns the most-active boards in the given team since
// the given unix-millisecond timestamp. boardIDs is the ACL set of boards
// the requesting user can access; only boards in that set are eligible.
func (s *Store) TopBoardsForTeam(ctx context.Context, teamID string, boardIDs []string, since int64, page, perPage int) (*insights.TopBoardList, error) {
	if len(boardIDs) == 0 {
		return &insights.TopBoardList{Items: []*insights.TopBoard{}}, nil
	}
	offset := page * perPage
	limit := perPage + 1

	rows, err := s.replica.QueryContext(ctx, teamBoardsSQLTemplate,
		time.UnixMilli(since).Format(time.RFC3339Nano),
		teamID,
		pq.Array(boardIDs),
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("TopBoardsForTeam query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanBoardList(rows, perPage)
}

// TopBoardsForUser returns the most-active boards in the given team
// restricted to those created by `userID` or that `userID` has touched
// (per active_users aggregation).
func (s *Store) TopBoardsForUser(ctx context.Context, teamID, userID string, boardIDs []string, since int64, page, perPage int) (*insights.TopBoardList, error) {
	if len(boardIDs) == 0 {
		return &insights.TopBoardList{Items: []*insights.TopBoard{}}, nil
	}
	offset := page * perPage
	limit := perPage + 1

	q := fmt.Sprintf(userBoardsSQLTemplate, topBoardsCountSelector)
	rows, err := s.replica.QueryContext(ctx, q,
		time.UnixMilli(since).Format(time.RFC3339Nano),
		teamID,
		pq.Array(boardIDs),
		userID,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("TopBoardsForUser query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanBoardList(rows, perPage)
}

// BoardIDsForUserInTeam returns the IDs of every focalboard board the user
// can see in the given team. Ported from
// `searchBoardsForUserInTeam` in mattermost-plugin-boards
// (server/services/store/sqlstore/board.go) at commit c8e729b6^, with
// MySQL/SQLite branches stripped and the term-search clause omitted.
//
// The result is the union of:
//   - All open boards in the team (boards.type = 'O').
//   - Boards where the user is a direct member (focalboard_board_members).
//   - Boards in channels the user is a member of (ChannelMembers).
//
// Templates (boards.is_template = true) and soft-deleted boards
// (boards.delete_at != 0) are excluded.
const boardIDsForUserInTeamSQL = `
	SELECT DISTINCT b.id FROM (
		SELECT id FROM focalboard_boards
		WHERE team_id = $1 AND type = 'O'
		  AND COALESCE(is_template, false) = false
		  AND COALESCE(delete_at, 0) = 0

		UNION

		SELECT b.id FROM focalboard_boards b
		JOIN focalboard_board_members bm ON b.id = bm.board_id
		WHERE bm.user_id = $2 AND b.team_id = $1
		  AND COALESCE(b.is_template, false) = false
		  AND COALESCE(b.delete_at, 0) = 0

		UNION

		SELECT b.id FROM focalboard_boards b
		JOIN channelmembers cm ON cm.channelid = b.channel_id
		WHERE cm.userid = $2 AND b.team_id = $1
		  AND COALESCE(b.is_template, false) = false
		  AND COALESCE(b.delete_at, 0) = 0
	) AS b
`

// BoardIDsForUserInTeam returns the IDs of every focalboard board the
// given user can see in the team.
func (s *Store) BoardIDsForUserInTeam(ctx context.Context, userID, teamID string) ([]string, error) {
	rows, err := s.replica.QueryContext(ctx, boardIDsForUserInTeamSQL, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("BoardIDsForUserInTeam: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanBoardList(rows *sql.Rows, perPage int) (*insights.TopBoardList, error) {
	items := make([]*insights.TopBoard, 0, perPage+1)
	for rows.Next() {
		var b insights.TopBoard
		var activeUsers sql.NullString
		var icon sql.NullString
		var activity int64
		if err := rows.Scan(&b.BoardID, &b.Title, &icon, &activity, &activeUsers, &b.CreatedBy); err != nil {
			return nil, err
		}
		if icon.Valid {
			b.Icon = icon.String
		}
		b.ActivityCount = strconv.FormatInt(activity, 10)
		if activeUsers.Valid && activeUsers.String != "" {
			b.ActiveUsers = strings.Split(activeUsers.String, ",")
		} else {
			b.ActiveUsers = []string{}
		}
		items = append(items, &b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	page, hasNext := insights.Paginate(items, perPage)
	return &insights.TopBoardList{
		ListData: insights.ListData{HasNext: hasNext},
		Items:    page,
	}, nil
}
