package insights

import (
	"github.com/mattermost/mattermost/server/public/model"
)

const (
	// TimeRangeToday is retired: "since midnight" is a moving, partial
	// window that a periodically-rebuilt snapshot cannot answer coherently.
	// TimeRange1Day replaces it with a fixed, closed one-day window, which
	// the snapshot handles exactly as it does the longer ranges.
	TimeRangeToday = "today"
	TimeRange1Day  = "1_day"
	TimeRange7Day  = "7_day"
	TimeRange28Day = "28_day"

	PostsByHour = "hour"
	PostsByDay  = "day"
)

type Opts struct {
	StartUnixMilli int64
	Page           int
	PerPage        int
}

type ListData struct {
	HasNext bool `json:"has_next"`

	// NotAvailable marks an insight that is switched off rather than empty,
	// so the webapp can tell "this card is disabled" apart from "this card
	// found no data". Omitted from the payload unless true.
	NotAvailable bool `json:"not_available,omitempty"`
}

// Top Reactions

type TopReactionList struct {
	ListData
	Items []*TopReaction `json:"items"`
}

type TopReaction struct {
	EmojiName string `json:"emoji_name"`
	Count     int64  `json:"count"`
}

// Top Channels

type TopChannelList struct {
	ListData
	Items               []*TopChannel              `json:"items"`
	PostCountByDuration ChannelPostCountByDuration `json:"channel_post_counts_by_duration"`
}

// ChannelIDs returns the IDs of every channel in the list, in order.
func (l *TopChannelList) ChannelIDs() []string {
	if l == nil {
		return nil
	}
	ids := make([]string, 0, len(l.Items))
	for _, item := range l.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

type TopChannel struct {
	ID           string            `json:"id"`
	Type         model.ChannelType `json:"type"`
	DisplayName  string            `json:"display_name"`
	Name         string            `json:"name"`
	TeamID       string            `json:"team_id"`
	MessageCount int64             `json:"message_count"`
}

// Channel activity / governance

// ChannelActivity is one channel's aggregate for a time window, before any
// per-user visibility filtering has been applied. Rows are cached team-wide
// and filtered at read time, so this deliberately carries no field that
// depends on who is asking.
//
// The same row backs three surfaces: Top Channels (sorted by MessageCount
// descending), Top Inactive Channels (ascending), and the channel-governance
// table (untruncated, with the metadata columns shown).
type ChannelActivity struct {
	ID          string            `json:"id"`
	Type        model.ChannelType `json:"type"`
	DisplayName string            `json:"display_name"`
	Name        string            `json:"name"`

	// Governance metadata. Purpose and Header are the only free-text labels
	// Mattermost channels carry — there is no tag field, and sidebar
	// categories are per-user rather than per-channel.
	Purpose  string `json:"purpose"`
	Header   string `json:"header"`
	CreateAt int64  `json:"create_at"`

	// LastPostAt is all-time, read straight off Channels — "when was this
	// channel last touched at all", which is what the governance table wants
	// for a channel that has been silent for months.
	//
	// LastPostInWindow is the newest post inside the window, or 0 if there
	// were none. Top Inactive Channels reports this one: its deprecated query
	// derived LastActivityAt from max(Posts.CreateAt) over the windowed join,
	// so a channel with no recent posts showed 0 rather than its real age.
	LastPostAt       int64 `json:"last_post_at"`
	LastPostInWindow int64 `json:"last_post_in_window"`

	// MessageCount and ActivePosters cover the window; MemberCount is
	// current. A high MemberCount against a zero MessageCount is the
	// clearest "abandoned channel" signal in the table.
	MessageCount  int64 `json:"message_count"`
	ActivePosters int64 `json:"active_posters"`
	MemberCount   int64 `json:"member_count"`
}

// ChannelGovernanceList is the channel-governance table's payload: a page of
// channels plus a summary describing the whole team, not the page.
type ChannelGovernanceList struct {
	ListData
	Items   []*ChannelActivity       `json:"items"`
	Summary ChannelGovernanceSummary `json:"summary"`

	// GeneratedAt is when the underlying snapshot was computed, in unix
	// milliseconds, or 0 if it was built during this request. The UI shows it
	// because the numbers are up to a day old by design.
	GeneratedAt int64 `json:"generated_at"`
}

// ChannelGovernanceSummary answers "have we labelled our channels
// effectively" over the full visible set. Computed server-side because the
// client only ever holds one page.
type ChannelGovernanceSummary struct {
	TotalChannels int64 `json:"total_channels"`

	// ActiveChannels had at least one non-integration post in the window.
	// TotalChannels minus this is the size of the cleanup backlog.
	ActiveChannels int64 `json:"active_channels"`

	// WithPurpose and WithHeader are the labelling-coverage numerators.
	// Mattermost channels carry no tags and sidebar categories are per-user,
	// so these two free-text fields are the only labelling signal available.
	WithPurpose int64 `json:"with_purpose"`
	WithHeader  int64 `json:"with_header"`

	// MatchingChannels is how many rows survived the search and filter. The
	// other counts deliberately describe the whole team regardless.
	MatchingChannels int64 `json:"matching_channels"`
}

// Top Inactive Channels

type TopInactiveChannelList struct {
	ListData
	Items []*TopInactiveChannel `json:"items"`
}

type TopInactiveChannel struct {
	ID             string            `json:"id"`
	Type           model.ChannelType `json:"type"`
	DisplayName    string            `json:"display_name"`
	Name           string            `json:"name"`
	LastActivityAt int64             `json:"last_activity_at"`
	Participants   model.StringArray `json:"participants"`
	MessageCount   int64             `json:"-"`
}

// Top Threads

type TopThreadList struct {
	ListData
	Items []*TopThread `json:"items"`
}

type TopThread struct {
	PostID          string            `json:"-"`
	ReplyCount      int64             `json:"-"`
	ChannelID       string            `json:"channel_id"`
	DisplayName     string            `json:"channel_display_name"`
	Name            string            `json:"channel_name"`
	Participants    model.StringArray `json:"participants"`
	UserID          string            `json:"-"`
	UserInformation *UserInformation  `json:"user_information"`
	Post            *model.Post       `json:"post"`
}

type UserInformation struct {
	ID                string `json:"id"`
	LastPictureUpdate int64  `json:"last_picture_update"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	NickName          string `json:"nickname"`
	Username          string `json:"username"`
}

// Top DMs

type DMUserInformation struct {
	UserInformation
	Position string `json:"position"`
}

type TopDM struct {
	MessageCount         int64              `json:"post_count"`
	OutgoingMessageCount int64              `json:"outgoing_message_count"`
	Participants         string             `json:"-"`
	ChannelID            string             `json:"-"`
	SecondParticipant    *DMUserInformation `json:"second_participant"`
}

type TopDMList struct {
	ListData
	Items []*TopDM `json:"items"`
}

// New Team Members

type NewTeamMembersList struct {
	ListData
	Items      []*NewTeamMember `json:"items"`
	TotalCount int64            `json:"total_count"`
}

type NewTeamMember struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	FirstName         string `json:"first_name"`
	LastName          string `json:"last_name"`
	Position          string `json:"position"`
	Nickname          string `json:"nickname"`
	LastPictureUpdate int64  `json:"last_picture_update,omitempty"`
	CreateAt          int64  `json:"create_at"`
}

// Top Playbooks
//
// Ported from mattermost-plugin-playbooks's `PlaybookInsight`
// (server/app/playbook.go in plugin HEAD `da4c39fc`). Field names match
// the deprecated JSON shape so any downstream consumer of the original
// Playbooks-plugin endpoint receives an identical response.
//
// Why this lives in this plugin and not in the Playbooks plugin: the
// Playbooks plugin's `licenseAndGuestCheck` only accepts `professional`
// or `enterprise` SKUs, so on an `advanced` (Enterprise Advanced) license
// the team-scope endpoint always returns 500. Reimplementing the SQL
// here lets our plugin use its own Professional+ gate (which accepts
// `advanced` correctly via `model.MinimumProfessionalLicense`).

type TopPlaybookList struct {
	ListData
	Items []*TopPlaybook `json:"items"`
}

type TopPlaybook struct {
	PlaybookID string `json:"playbook_id"`
	NumRuns    int64  `json:"num_runs"`
	Title      string `json:"title"`
	LastRunAt  int64  `json:"last_run_at"`
}

// Top Boards
//
// Ported from mattermost-plugin-boards's `BoardInsight` (deleted in
// commit c8e729b6, June 2024). Field names match the deprecated JSON shape
// so that any downstream consumer of the original Insights endpoint
// receives identical-looking responses.

type TopBoardList struct {
	ListData
	Items []*TopBoard `json:"items"`
}

type TopBoard struct {
	BoardID       string            `json:"boardID"`
	Icon          string            `json:"icon"`
	Title         string            `json:"title"`
	ActivityCount string            `json:"activityCount"`
	ActiveUsers   model.StringArray `json:"activeUsers"`
	CreatedBy     string            `json:"createdBy"`
}

// Post-count chart

// DurationPostCount is a single (channel, time-bucket, count) row from the
// post-count-by-duration query that backs Top Channels' chart.
type DurationPostCount struct {
	ChannelID string
	Duration  string // ISO8601 date or date-hour string
	PostCount int
}

// ChannelPostCountByDuration maps an ISO8601 duration string to a per-channel
// post-count map. Example (grouped by day):
//
//	{
//	  "2009-11-11": {"chan_a": 90, "chan_b": 201},
//	  "2009-11-12": {"chan_a": 45, "chan_b": 68},
//	}
type ChannelPostCountByDuration map[string]map[string]int
