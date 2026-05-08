package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// SQL ported from
// mattermost-plugin-playbooks/server/sqlstore/playbook.go's
// `insightsQueryBuilder` (HEAD `da4c39fc` on 2026-05-08). Why this
// lives in our plugin and not in the Playbooks plugin: the Playbooks
// plugin's `licenseAndGuestCheck` only accepts `professional` or
// `enterprise` SKUs, so on `advanced` (Enterprise Advanced) licenses the
// Playbooks-plugin endpoint always returns 500. By querying the
// IR_Playbook / IR_Incident / IR_PlaybookMember tables ourselves, our
// own Professional+ gate (which accepts `advanced` correctly via
// `model.MinimumProfessionalLicense`) governs access.
//
// Tables are NOT prefixed (the Playbooks plugin uses bare names like
// `IR_Playbook`, no `playbooks_` prefix).

// teamPlaybooksSQL: top playbooks for a whole team. The user can see
// playbooks they're a member of OR public playbooks in that team.
const teamPlaybooksSQL = `
	SELECT p.ID AS playbook_id,
	       p.Title AS title,
	       count(i.ID) AS num_runs,
	       COALESCE(MAX(i.CreateAt), 0) AS last_run_at
	FROM IR_Playbook AS p
	LEFT JOIN IR_Incident AS i ON p.ID = i.PlaybookID
	WHERE p.TeamID = $1
	  AND (
	      EXISTS(SELECT 1 FROM IR_PlaybookMember AS pm WHERE pm.PlaybookID = p.ID AND pm.MemberID = $2)
	      OR p.Public = true
	  )
	  AND i.CreateAt >= $3
	GROUP BY p.ID
	ORDER BY num_runs DESC
	LIMIT $4 OFFSET $5
`

// userPlaybooksSQL: my-scope variant — only playbooks the user is
// directly a member of, regardless of public/private.
const userPlaybooksSQL = `
	SELECT p.ID AS playbook_id,
	       p.Title AS title,
	       count(i.ID) AS num_runs,
	       COALESCE(MAX(i.CreateAt), 0) AS last_run_at
	FROM IR_Playbook AS p
	LEFT JOIN IR_Incident AS i ON p.ID = i.PlaybookID
	WHERE p.TeamID = $1
	  AND EXISTS(SELECT 1 FROM IR_PlaybookMember AS pm WHERE pm.PlaybookID = p.ID AND pm.MemberID = $2)
	  AND i.CreateAt >= $3
	GROUP BY p.ID
	ORDER BY num_runs DESC
	LIMIT $4 OFFSET $5
`

// TopPlaybooksForTeam returns the most-active playbooks in the team for
// the given user (filtered to playbooks the user can see).
func (s *Store) TopPlaybooksForTeam(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopPlaybookList, error) {
	offset := page * perPage
	limit := perPage + 1

	rows, err := s.replica.QueryContext(ctx, teamPlaybooksSQL, teamID, userID, since, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("TopPlaybooksForTeam query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanPlaybookList(rows, perPage)
}

// TopPlaybooksForUser returns the most-active playbooks the user is a
// direct member of.
func (s *Store) TopPlaybooksForUser(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopPlaybookList, error) {
	offset := page * perPage
	limit := perPage + 1

	rows, err := s.replica.QueryContext(ctx, userPlaybooksSQL, teamID, userID, since, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("TopPlaybooksForUser query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanPlaybookList(rows, perPage)
}

func scanPlaybookList(rows *sql.Rows, perPage int) (*insights.TopPlaybookList, error) {
	items := make([]*insights.TopPlaybook, 0, perPage+1)
	for rows.Next() {
		var p insights.TopPlaybook
		if err := rows.Scan(&p.PlaybookID, &p.Title, &p.NumRuns, &p.LastRunAt); err != nil {
			return nil, err
		}
		items = append(items, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	page, hasNext := insights.Paginate(items, perPage)
	return &insights.TopPlaybookList{
		ListData: insights.ListData{HasNext: hasNext},
		Items:    page,
	}, nil
}
