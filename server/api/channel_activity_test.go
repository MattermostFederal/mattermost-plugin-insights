package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/api/apitest"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/cache"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

const (
	otherUserID = "user2aaaaaaaaaaaaaaaaaaaaa"
	pubChanID   = "pub0aaaaaaaaaaaaaaaaaaaaaa"
	privMineID  = "prv0aaaaaaaaaaaaaaaaaaaaaa"
	privTheirID = "prv1aaaaaaaaaaaaaaaaaaaaaa"
)

// activityAPI wires two team members with different private-channel access
// against one shared team-wide aggregate.
func activityAPI() (*API, *apitest.StoreStub) {
	auth := &apitest.AuthStub{
		Users: map[string]*model.User{
			testUserID:  newRegularUser(testUserID),
			otherUserID: newRegularUser(otherUserID),
		},
		License: professionalLicense(),
		TeamPerms: map[string]map[string]bool{
			testUserID:  {testTeamID: true},
			otherUserID: {testTeamID: true},
		},
	}
	store := &apitest.StoreStub{
		ChannelActivityResult: []*insights.ChannelActivity{
			{ID: pubChanID, Name: "public", Type: model.ChannelTypeOpen, MessageCount: 10, CreateAt: 1},
			{ID: privMineID, Name: "mine", Type: model.ChannelTypePrivate, MessageCount: 50, CreateAt: 1},
			{ID: privTheirID, Name: "theirs", Type: model.ChannelTypePrivate, MessageCount: 99, CreateAt: 1},
		},
		PrivateChannelIDsResult: map[string][]string{
			testUserID:  {privMineID},
			otherUserID: {privTheirID},
		},
	}
	return New(auth, &apitest.DirectoryStub{}, store), store
}

func getChannels(t *testing.T, api *API, userID string) *insights.TopChannelList {
	t.Helper()
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=7_day", userID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	return decodeChannels(t, w)
}

// The core of the snapshot design: the expensive aggregate runs once for the
// team no matter how many members load the page.
func TestChannelActivity_aggregateRunsOncePerTeamAndRange(t *testing.T) {
	api, store := activityAPI()

	for range 4 {
		getChannels(t, api, testUserID)
	}
	getChannels(t, api, otherUserID)

	if n := len(store.ChannelActivityCalls); n != 1 {
		t.Errorf("ChannelActivityForTeam ran %d times across 5 requests; want 1", n)
	}
	// The per-user visibility lookup is deliberately NOT cached — it is what
	// keeps two members from seeing each other's private channels.
	if n := len(store.PrivateChannelIDsCalls); n != 5 {
		t.Errorf("PrivateChannelIDsForUser ran %d times; want 5 (once per request)", n)
	}
}

func TestChannelActivity_separateEntryPerTimeRange(t *testing.T) {
	api, store := activityAPI()

	for _, tr := range []string{"7_day", "28_day", "7_day", "28_day"} {
		w := httptest.NewRecorder()
		api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range="+tr, testUserID))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", tr, w.Code)
		}
	}
	if n := len(store.ChannelActivityCalls); n != 2 {
		t.Errorf("ChannelActivityForTeam ran %d times; want 2 (one per range)", n)
	}
}

// Two members of the same team share one cached aggregate but must not see
// each other's private channels.
func TestChannelActivity_privateChannelsFilteredPerUser(t *testing.T) {
	api, _ := activityAPI()

	mine := getChannels(t, api, testUserID)
	theirs := getChannels(t, api, otherUserID)

	names := func(b *insights.TopChannelList) map[string]bool {
		out := map[string]bool{}
		for _, i := range b.Items {
			out[i.Name] = true
		}
		return out
	}

	m, t2 := names(mine), names(theirs)
	if !m["public"] || !t2["public"] {
		t.Error("both users should see the public channel")
	}
	if !m["mine"] || m["theirs"] {
		t.Errorf("user1 saw %v; want public+mine only", m)
	}
	if !t2["theirs"] || t2["mine"] {
		t.Errorf("user2 saw %v; want public+theirs only", t2)
	}
}

// Filtering must copy rather than reslice: the cached backing array is handed
// to every concurrent reader on the team, so one user's filter must not be
// visible to the next.
func TestChannelActivity_filteringDoesNotCorruptTheCachedSlice(t *testing.T) {
	api, _ := activityAPI()

	getChannels(t, api, otherUserID)
	second := getChannels(t, api, testUserID)

	found := map[string]bool{}
	for _, i := range second.Items {
		found[i.Name] = true
	}
	if !found["public"] || !found["mine"] {
		t.Errorf("second reader saw %v; the cached slice was mutated by the first", found)
	}
}

func TestChannelActivity_sortsDescendingForTopChannels(t *testing.T) {
	api, _ := activityAPI()
	body := getChannels(t, api, testUserID)

	if len(body.Items) != 2 {
		t.Fatalf("got %d items; want 2", len(body.Items))
	}
	if body.Items[0].Name != "mine" || body.Items[1].Name != "public" {
		t.Errorf("order = %s, %s; want mine (50) before public (10)", body.Items[0].Name, body.Items[1].Name)
	}
}

func TestChannelActivity_sortsAscendingForInactiveChannels(t *testing.T) {
	api, _ := activityAPI()

	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/inactive_channels?time_range=7_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}
	body := decodeInactiveChannels(t, w)
	if len(body.Items) != 2 {
		t.Fatalf("got %d items; want 2", len(body.Items))
	}
	if body.Items[0].Name != "public" || body.Items[1].Name != "mine" {
		t.Errorf("order = %s, %s; want public (10) before mine (50)", body.Items[0].Name, body.Items[1].Name)
	}
}

// A channel created inside the window cannot fairly be called inactive over
// that window — the deprecated query's `Channels.CreateAt < since` filter.
func TestChannelActivity_inactiveExcludesChannelsNewerThanWindow(t *testing.T) {
	api, store := activityAPI()
	store.ChannelActivityResult = append(store.ChannelActivityResult, &insights.ChannelActivity{
		ID: "brandnewaaaaaaaaaaaaaaaaaa", Name: "brand-new", Type: model.ChannelTypeOpen,
		MessageCount: 0,
		// Far in the future relative to any window start.
		CreateAt: 1 << 62,
	})

	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/inactive_channels?time_range=7_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", w.Code, w.Body.String())
	}
	for _, item := range decodeInactiveChannels(t, w).Items {
		if item.Name == "brand-new" {
			t.Error("a channel created inside the window was reported as inactive")
		}
	}
}

func TestChannelActivity_paginates(t *testing.T) {
	api, _ := activityAPI()

	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=7_day&per_page=1&page=0", testUserID))
	first := decodeChannels(t, w)
	if len(first.Items) != 1 || first.Items[0].Name != "mine" {
		t.Fatalf("page 0 = %#v", first.Items)
	}
	if !first.HasNext {
		t.Error("page 0 should report has_next")
	}

	w = httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=7_day&per_page=1&page=1", testUserID))
	second := decodeChannels(t, w)
	if len(second.Items) != 1 || second.Items[0].Name != "public" {
		t.Fatalf("page 1 = %#v", second.Items)
	}
	if second.HasNext {
		t.Error("page 1 is the last page; has_next should be false")
	}

	w = httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=7_day&per_page=1&page=9", testUserID))
	if items := decodeChannels(t, w).Items; len(items) != 0 {
		t.Errorf("page past the end returned %d items; want 0", len(items))
	}
}

// Every range is cached for a full day. That is only sound because the
// windows are closed: a snapshot of a period that has already ended cannot go
// stale within the day. See insights.Window.
func TestChannelActivity_allRangesUseTheDailyTTL(t *testing.T) {
	if cache.DefaultTTL != 24*time.Hour {
		t.Errorf("DefaultTTL = %v; want 24h", cache.DefaultTTL)
	}
}

func TestChannelActivity_oneDayRangeIsServed(t *testing.T) {
	api, store := activityAPI()

	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=1_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	if len(store.ChannelActivityCalls) != 1 {
		t.Fatalf("expected one aggregate call, got %d", len(store.ChannelActivityCalls))
	}

	// Its own cache entry, separate from the longer windows.
	w = httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=7_day", testUserID))
	if n := len(store.ChannelActivityCalls); n != 2 {
		t.Errorf("aggregate ran %d times across two ranges; want 2", n)
	}
}

// "today" stays rejected: a moving since-midnight window is what the snapshot
// genuinely cannot answer, as distinct from a fixed one-day window.
func TestChannelActivity_todayStillRejected(t *testing.T) {
	api, _ := activityAPI()
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=today", testUserID))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want 400 for 'today'", w.Code)
	}
}
