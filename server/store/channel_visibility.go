// Channel visibility — the read-time half of the cached-snapshot design.
//
// ChannelActivityForTeam deliberately returns every channel in the team,
// unfiltered, so one cached slice can serve everyone on it. That slice is not
// safe to hand back as-is: private channels are only visible to their members.
// Callers intersect it with this query's result before rendering.
//
// Public channels are not enumerated here. Every member of a team can see the
// team's public channels without a ChannelMembers row, so the caller's rule is
// "keep the row if it is public, or if its ID is in this set" — which keeps
// the query to the small, indexed membership lookup.

package store

import (
	"context"
)

const privateChannelIDsForUserSQL = `
SELECT Channels.Id
FROM Channels
JOIN ChannelMembers ON ChannelMembers.ChannelId = Channels.Id
WHERE ChannelMembers.UserId = $1
	AND Channels.TeamId = $2
	AND Channels.DeleteAt = 0
	AND Channels.Type = 'P'
`

// PrivateChannelIDsForUser returns the IDs of the team's private channels that
// the given user is a member of. Archived channels are excluded, as are
// memberships in other teams.
//
// Returns an empty slice rather than nil when the user has no private
// memberships, so callers can range over the result unconditionally.
func (s *Store) PrivateChannelIDsForUser(ctx context.Context, userID, teamID string) ([]string, error) {
	rows, err := s.replica.QueryContext(ctx, privateChannelIDsForUserSQL, userID, teamID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
