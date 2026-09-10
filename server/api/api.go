package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/cache"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// Storer is the subset of methods API handlers call against the database
// store. Concrete implementation lives in server/store; tests substitute
// stubs.
type Storer interface {
	TopReactionsForUserSince(ctx context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopReactionList, error)
	TopReactionsForTeamSince(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopReactionList, error)
	TopThreadsForUserSince(ctx context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopThreadList, error)
	TopThreadsForTeamSince(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopThreadList, error)
	NewTeamMembersSince(ctx context.Context, teamID string, w insights.Window, page, perPage int, showFullName bool) (*insights.NewTeamMembersList, error)
	TopChannelsForUserSince(ctx context.Context, userID, teamID string, start, end int64, page, perPage int) (*insights.TopChannelList, error)
	TopChannelsForTeamSince(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopChannelList, error)

	// ChannelActivityForTeam is the cached team-wide aggregate behind Top
	// Channels, Top Inactive Channels, and the governance table. It takes no
	// userID and no pagination so one result can be shared across the team;
	// PrivateChannelIDsForUser supplies the per-request visibility filter.
	ChannelActivityForTeam(ctx context.Context, teamID string, w insights.Window) ([]*insights.ChannelActivity, error)
	PrivateChannelIDsForUser(ctx context.Context, userID, teamID string) ([]string, error)

	// AttachInactiveChannelParticipants fills in Participants for the rows
	// on the current page. Deliberately not cached: it is bounded by
	// per_page and keyed to the page's channel ids, so it stays small.
	AttachInactiveChannelParticipants(ctx context.Context, channels []*insights.TopInactiveChannel) error
	TopInactiveChannelsForUserSince(ctx context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopInactiveChannelList, error)
	TopInactiveChannelsForTeamSince(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopInactiveChannelList, error)
	TopDMsForUserSince(ctx context.Context, userID string, since int64, page, perPage int) (*insights.TopDMList, error)
	OutgoingDMCounts(ctx context.Context, userID string, channelIDs []string, since int64) (map[string]int64, error)
	PostCountsByDuration(ctx context.Context, channelIDs []string, startUnixMillis, endUnixMillis int64, userID, grouping, location string) ([]*insights.DurationPostCount, error)

	// Top Boards: BoardIDsForUserInTeam computes the user's accessible
	// board ACL set; TopBoardsForTeam / TopBoardsForUser run the
	// activity aggregation against that set.
	BoardIDsForUserInTeam(ctx context.Context, userID, teamID string) ([]string, error)
	TopBoardsForTeam(ctx context.Context, teamID string, boardIDs []string, since int64, page, perPage int) (*insights.TopBoardList, error)
	TopBoardsForUser(ctx context.Context, teamID, userID string, boardIDs []string, since int64, page, perPage int) (*insights.TopBoardList, error)

	// Top Playbooks: queries IR_Playbook / IR_Incident /
	// IR_PlaybookMember directly so this plugin's own license gate
	// (which accepts Enterprise Advanced) governs access, instead of
	// the Playbooks plugin's gate (which doesn't).
	TopPlaybooksForTeam(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopPlaybookList, error)
	TopPlaybooksForUser(ctx context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopPlaybookList, error)
}

// AuthProvider is the subset of pluginapi.Client methods the auth gates need.
// Tests substitute a fake; production wraps a *pluginapi.Client.
type AuthProvider interface {
	GetUser(userID string) (*model.User, error)
	HasPermissionToTeam(userID, teamID string, perm *model.Permission) bool
	GetLicense() *model.License
}

// Telemetry records insights-related events. Tests substitute a no-op /
// capturing fake; production wraps the host's structured-log helper so
// admins can scrape the events from Mattermost server logs and forward
// them to their own analytics pipeline (mirrors what the deprecated
// `trackEvent('insights', '<event>')` did via the host's RudderStack
// integration, without bundling RudderStack credentials in the plugin).
type Telemetry interface {
	TrackInsightsEvent(event string, properties map[string]any)
}

// Directory is the subset of pluginapi.Client used to hydrate insight rows
// with related Mattermost objects (user profiles, root posts). Kept separate
// from AuthProvider so test fakes can target only the lookup surface.
type Directory interface {
	ListUsersByIDs(userIDs []string) ([]*model.User, error)
	GetPost(postID string) (*model.Post, error)

	// ShowFullName returns the server's PrivacySettings.ShowFullName.
	// When false, non-admins should not see other users' first/last names.
	ShowFullName() bool
}

// API is the HTTP surface of the Insights plugin. All routes are mounted
// under /plugins/insights/api/v1/...; the Mattermost plugin host strips the
// /plugins/insights/ prefix before invoking ServeHTTP.
type API struct {
	auth      AuthProvider
	directory Directory
	store     Storer
	telemetry Telemetry
	router    *mux.Router

	// cache holds the daily team snapshots. Entries are keyed by team, time
	// range, and the window's date — never by user — so the expensive
	// aggregation runs once a day rather than once per page load, and rolls
	// over at midnight UTC rather than 24h after it happened to be built.
	cache *cache.Cache
}

// EnablePersonalInsights controls whether the user-scoped ("My") insight
// routes are served. Personal insights are disabled while the plugin moves
// team insights onto a once-daily server-wide snapshot: the My-scope queries
// run per-request and carry the performance problems catalogued in
// INSIGHTS_REFERENCE.md §3, and there is no per-user equivalent of the
// snapshot.
//
// Consequence: team routes require a Professional+ license, so with this
// off the plugin serves nothing on unlicensed, Starter, or non-enterprise
// builds. That is accepted for now and expected to change.
const EnablePersonalInsights = false

// EnableBoardsAndPlaybooks controls whether the Top Boards and Top Playbooks
// insights run their queries. Both are stubbed off: they read tables owned by
// other plugins (focalboard_*, IR_*), which 500 outright when that plugin is
// absent, and neither is worth carrying onto the daily snapshot.
//
// The routes stay registered and keep their auth gates so the gates matrix
// stays uniform; the handlers return an empty list marked NotAvailable
// instead of querying.
const EnableBoardsAndPlaybooks = false

// EnableReactionsAndThreads controls whether Top Reactions and Top Threads
// run their queries. Both are off for 1.0.
//
// The requirement is that team insights are "cached server wide once a day"
// with no live aggregations on page load. Neither of these can meet it
// cheaply: both scope private channels per requester (INSIGHTS_REFERENCE.md
// §2.1, §2.3), so a shared snapshot needs channel-grain entries and a
// read-time sum — for reactions that means a (channel, emoji) entry, the
// largest payload of any insight. Top Threads additionally carries the §3.1
// N+1 hydration, which needs either stale post content or a new batched
// lookup.
//
// Switching them off satisfies the requirement immediately rather than after
// the two most expensive pieces of work left, and neither card appears in any
// stated requirement. Re-enable by flipping this once there is time to put
// them on the snapshot properly.
const EnableReactionsAndThreads = false

// New builds the API and wires every route.
func New(auth AuthProvider, directory Directory, st Storer) *API {
	return NewWithTelemetry(auth, nil, directory, st, noopTelemetry{})
}

// NewWithTelemetry builds the API with a telemetry recorder. Most callers
// (including tests) use New; production calls NewWithTelemetry from
// FromPluginAPI so events land in the host server logs.
func NewWithTelemetry(auth AuthProvider, _ /*reserved*/ any, directory Directory, st Storer, tel Telemetry) *API {
	if tel == nil {
		tel = noopTelemetry{}
	}
	a := &API{
		auth:      auth,
		directory: directory,
		store:     st,
		telemetry: tel,
		cache:     cache.New(cache.Options{}),
	}

	r := mux.NewRouter()
	v1 := r.PathPrefix("/api/v1").Subrouter()

	v1.HandleFunc("/healthz", a.requireUser(a.handleHealthz)).Methods(http.MethodGet)

	// Team-scoped insights (require Professional+ license + view_team).
	v1.HandleFunc("/teams/{team_id}/top/reactions", a.requireUser(a.handleTopReactionsForTeam)).Methods(http.MethodGet)
	v1.HandleFunc("/teams/{team_id}/top/channels", a.requireUser(a.handleTopChannelsForTeam)).Methods(http.MethodGet)
	v1.HandleFunc("/teams/{team_id}/top/threads", a.requireUser(a.handleTopThreadsForTeam)).Methods(http.MethodGet)
	v1.HandleFunc("/teams/{team_id}/top/inactive_channels", a.requireUser(a.handleTopInactiveChannelsForTeam)).Methods(http.MethodGet)
	v1.HandleFunc("/teams/{team_id}/top/team_members", a.requireUser(a.handleNewTeamMembers)).Methods(http.MethodGet)
	v1.HandleFunc("/teams/{team_id}/top/boards", a.requireUser(a.handleTopBoardsForTeam)).Methods(http.MethodGet)
	v1.HandleFunc("/teams/{team_id}/top/playbooks", a.requireUser(a.handleTopPlaybooksForTeam)).Methods(http.MethodGet)

	// Channel governance — every channel in the team with its activity and
	// metadata, rather than a top-N. Same gates as the routes above.
	v1.HandleFunc("/teams/{team_id}/channel_activity", a.requireUser(a.handleChannelGovernance)).Methods(http.MethodGet)

	// User-scoped insights (no license requirement).
	//
	// Disabled — see EnablePersonalInsights. The handlers and their store
	// queries are intentionally left in the tree so re-enabling is a
	// one-line change while the product decision is still open.
	if EnablePersonalInsights {
		v1.HandleFunc("/users/me/top/reactions", a.requireUser(a.handleTopReactionsForUser)).Methods(http.MethodGet)
		v1.HandleFunc("/users/me/top/channels", a.requireUser(a.handleTopChannelsForUser)).Methods(http.MethodGet)
		v1.HandleFunc("/users/me/top/threads", a.requireUser(a.handleTopThreadsForUser)).Methods(http.MethodGet)
		v1.HandleFunc("/users/me/top/dms", a.requireUser(a.handleTopDMsForUser)).Methods(http.MethodGet)
		v1.HandleFunc("/users/me/top/inactive_channels", a.requireUser(a.handleTopInactiveChannelsForUser)).Methods(http.MethodGet)
		v1.HandleFunc("/users/me/top/boards", a.requireUser(a.handleTopBoardsForUser)).Methods(http.MethodGet)
		v1.HandleFunc("/users/me/top/playbooks", a.requireUser(a.handleTopPlaybooksForUser)).Methods(http.MethodGet)
	}

	// Telemetry — receives `trackEvent('insights', '<event>', props?)`
	// posts from the webapp.
	v1.HandleFunc("/telemetry", a.requireUser(a.handleTelemetry)).Methods(http.MethodPost)

	r.NotFoundHandler = http.HandlerFunc(a.handleNotFound)
	r.MethodNotAllowedHandler = http.HandlerFunc(a.handleMethodNotAllowed)

	a.router = r
	return a
}

// FromPluginAPI builds an API around a real pluginapi.Client and store.
// Production code (server/plugin.go) calls this; tests build the API directly
// with stubbed AuthProvider / Directory / Storer.
func FromPluginAPI(client *pluginapi.Client, st Storer) *API {
	adapter := pluginAPIClient{client: client}
	return NewWithTelemetry(adapter, nil, adapter, st, pluginAPITelemetry{client: client})
}

// ServeHTTP delegates to the gorilla/mux router.
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) { a.router.ServeHTTP(w, r) }

func (a *API) handleHealthz(w http.ResponseWriter, _ *http.Request, _ string) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) handleNotFound(w http.ResponseWriter, _ *http.Request) {
	writeJSONError(w, http.StatusNotFound, "not found")
}

func (a *API) handleMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// pluginAPIClient adapts a *pluginapi.Client to both AuthProvider and
// Directory. One adapter, two interfaces.
type pluginAPIClient struct {
	client *pluginapi.Client
}

func (a pluginAPIClient) GetUser(userID string) (*model.User, error) {
	return a.client.User.Get(userID)
}

func (a pluginAPIClient) HasPermissionToTeam(userID, teamID string, perm *model.Permission) bool {
	return a.client.User.HasPermissionToTeam(userID, teamID, perm)
}

func (a pluginAPIClient) GetLicense() *model.License {
	return a.client.System.GetLicense()
}

func (a pluginAPIClient) ListUsersByIDs(userIDs []string) ([]*model.User, error) {
	return a.client.User.ListByUserIDs(userIDs)
}

func (a pluginAPIClient) GetPost(postID string) (*model.Post, error) {
	return a.client.Post.GetPost(postID)
}

func (a pluginAPIClient) ShowFullName() bool {
	cfg := a.client.Configuration.GetUnsanitizedConfig()
	if cfg == nil || cfg.PrivacySettings.ShowFullName == nil {
		return true
	}
	return *cfg.PrivacySettings.ShowFullName
}
