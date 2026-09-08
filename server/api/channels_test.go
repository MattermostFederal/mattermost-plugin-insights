package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/api/apitest"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

func decodeChannels(t *testing.T, w *httptest.ResponseRecorder) *insights.TopChannelList {
	t.Helper()
	var body insights.TopChannelList
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return &body
}

func TestAPI_TopChannelsForUser_unauthenticated(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	api := New(&apitest.AuthStub{}, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/me/top/channels?time_range=today", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", w.Code)
	}
}

func TestAPI_TopChannelsForUser_returnsItems(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	store := &apitest.StoreStub{
		TopChannelsForUserResult: &insights.TopChannelList{
			Items: []*insights.TopChannel{{ID: "ch1", Name: "general", DisplayName: "General", MessageCount: 12}},
		},
	}
	api := New(auth, &apitest.DirectoryStub{}, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/channels?time_range=7_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeChannels(t, w)
	if len(body.Items) != 1 || body.Items[0].Name != "general" {
		t.Fatalf("body items = %#v", body.Items)
	}
	// PostCountByDuration must be non-nil in the JSON envelope (clients
	// type it as a Record<string, Record<string, number>> and crash on null).
	if body.PostCountByDuration == nil {
		t.Errorf("PostCountByDuration must be initialized to {} for client consumption")
	}
}

func TestAPI_TopChannelsForTeam_noLicense(t *testing.T) {
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=today", testUserID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403", w.Code)
	}
}

func TestAPI_TopChannelsForTeam_notMember(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:   map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License: professionalLicense(),
	}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=today", testUserID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403 when not a team member", w.Code)
	}
}

func TestAPI_TopChannelsForUser_attachesChartData(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	store := &apitest.StoreStub{
		TopChannelsForUserResult: &insights.TopChannelList{
			Items: []*insights.TopChannel{{ID: "chA", Name: "alpha", MessageCount: 5}},
		},
		PostCountsByDurationResult: []*insights.DurationPostCount{
			// "today" => hour buckets, RFC3339 keys post-formatting
			{ChannelID: "chA", Duration: "2024-01-10T10", PostCount: 5},
		},
	}
	api := New(auth, &apitest.DirectoryStub{}, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/channels?time_range=today", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeChannels(t, w)
	if body.PostCountByDuration == nil {
		t.Fatalf("PostCountByDuration must be initialized")
	}
	// Today => hour buckets => map has keys formatted as RFC3339.
	if len(body.PostCountByDuration) == 0 {
		t.Errorf("expected at least one hour bucket; got empty map")
	}
	// store.PostCountsByDuration must have been called with the user filter
	// (this is a my-scope endpoint).
	if len(store.PostCountsByDurationCalls) != 1 {
		t.Fatalf("expected 1 PostCountsByDuration call; got %d", len(store.PostCountsByDurationCalls))
	}
	call := store.PostCountsByDurationCalls[0]
	if call.UserID != testUserID {
		t.Errorf("user-scope endpoint should pass user filter; got UserID=%q", call.UserID)
	}
	if call.Grouping != insights.PostsByHour {
		t.Errorf("today range should use hour grouping; got %q", call.Grouping)
	}
}

func TestAPI_TopChannelsForTeam_chartHasNoUserFilter(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	store := &apitest.StoreStub{
		TopChannelsForTeamResult: &insights.TopChannelList{
			Items: []*insights.TopChannel{{ID: "chT", Name: "team-channel", MessageCount: 9}},
		},
	}
	api := New(auth, &apitest.DirectoryStub{}, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=7_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	if len(store.PostCountsByDurationCalls) != 1 {
		t.Fatalf("expected 1 PostCountsByDuration call")
	}
	call := store.PostCountsByDurationCalls[0]
	if call.UserID != "" {
		t.Errorf("team-scope chart should not filter by user; got UserID=%q", call.UserID)
	}
	if call.Grouping != insights.PostsByDay {
		t.Errorf("7-day range should use day grouping; got %q", call.Grouping)
	}
}

func TestAPI_TopChannelsForTeam_returnsItems(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	store := &apitest.StoreStub{
		TopChannelsForTeamResult: &insights.TopChannelList{
			Items: []*insights.TopChannel{{ID: "ch1", Name: "general", MessageCount: 50}},
		},
	}
	api := New(auth, &apitest.DirectoryStub{}, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=today", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeChannels(t, w)
	if len(body.Items) != 1 || body.Items[0].Name != "general" {
		t.Fatalf("body items = %#v", body.Items)
	}
}
