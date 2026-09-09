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

// Auth-gate cases adapted from server/channels/api4/insights_test.go in
// commit 26617fcbdc (sub-tests "invalid time range", "invalid license", "not
// a member of team", etc.).

const (
	testUserID  = "user1aaaaaaaaaaaaaaaaaaaaa"
	testGuestID = "guest1aaaaaaaaaaaaaaaaaaaa"
	testTeamID  = "team0aaaaaaaaaaaaaaaaaaaaa"
)

func newAuthedRequest(method, path, userID string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	if userID != "" {
		r.Header.Set("Mattermost-User-Id", userID)
	}
	return r
}

func newRegularUser(id string) *model.User {
	return &model.User{Id: id, Roles: "system_user"}
}

func newGuestUser(id string) *model.User {
	return &model.User{Id: id, Roles: "system_guest"}
}

func professionalLicense() *model.License {
	return model.NewTestLicenseSKU(model.LicenseShortSkuProfessional)
}

func decodeReactions(t *testing.T, w *httptest.ResponseRecorder) *insights.TopReactionList {
	t.Helper()
	var body insights.TopReactionList
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return &body
}

func TestAPI_TopReactionsForUser_unauthenticated(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	api := New(&apitest.AuthStub{}, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/me/top/reactions?time_range=7_day", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", w.Code)
	}
}

func TestAPI_TopReactionsForUser_guest(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	auth := &apitest.AuthStub{Users: map[string]*model.User{testGuestID: newGuestUser(testGuestID)}}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/reactions?time_range=7_day", testGuestID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403 for guest", w.Code)
	}
}

func TestAPI_TopReactionsForUser_invalidTimeRange(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/reactions?time_range=7_days", testUserID))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want 400 for invalid time range", w.Code)
	}
}

func TestAPI_TopReactionsForUser_returnsItems(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	store := &apitest.StoreStub{
		TopReactionsForUserResult: &insights.TopReactionList{
			ListData: insights.ListData{HasNext: true},
			Items: []*insights.TopReaction{
				{EmojiName: "100", Count: 6},
				{EmojiName: "joy", Count: 5},
			},
		},
	}
	api := New(auth, &apitest.DirectoryStub{}, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/reactions?time_range=7_day&page=0&per_page=5", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeReactions(t, w)
	if !body.HasNext || len(body.Items) != 2 || body.Items[0].EmojiName != "100" {
		t.Fatalf("body = %#v", body)
	}
	if len(store.TopReactionsForUserCalls) != 1 {
		t.Fatalf("expected 1 store call, got %d", len(store.TopReactionsForUserCalls))
	}
	call := store.TopReactionsForUserCalls[0]
	if call.UserID != testUserID || call.Page != 0 || call.PerPage != 5 || call.TeamID != "" {
		t.Fatalf("call = %#v", call)
	}
}

func TestAPI_TopReactionsForUser_passesTeamFilter(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	store := &apitest.StoreStub{TopReactionsForUserResult: &insights.TopReactionList{}}
	api := New(auth, &apitest.DirectoryStub{}, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/reactions?time_range=7_day&team_id="+testTeamID, testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	if len(store.TopReactionsForUserCalls) != 1 || store.TopReactionsForUserCalls[0].TeamID != testTeamID {
		t.Fatalf("expected store call with team filter; got %#v", store.TopReactionsForUserCalls)
	}
}

func TestAPI_TopReactionsForTeam_noLicense(t *testing.T) {
	auth := &apitest.AuthStub{
		Users: map[string]*model.User{testUserID: newRegularUser(testUserID)},
		// no License set
	}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/reactions?time_range=7_day", testUserID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403 without Professional license", w.Code)
	}
}

func TestAPI_TopReactionsForTeam_notMember(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:   map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License: professionalLicense(),
		// no TeamPerms entry => HasPermissionToTeam returns false
	}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/reactions?time_range=7_day", testUserID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403 when user lacks view_team", w.Code)
	}
}

func TestAPI_TopReactionsForTeam_returnsItems(t *testing.T) {
	skipIfReactionsAndThreadsDisabled(t)
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	store := &apitest.StoreStub{
		TopReactionsForTeamResult: &insights.TopReactionList{
			Items: []*insights.TopReaction{{EmojiName: "tada", Count: 9}},
		},
	}
	api := New(auth, &apitest.DirectoryStub{}, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/reactions?time_range=7_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeReactions(t, w)
	if len(body.Items) != 1 || body.Items[0].EmojiName != "tada" {
		t.Fatalf("body = %#v", body)
	}
	if len(store.TopReactionsForTeamCalls) != 1 || store.TopReactionsForTeamCalls[0].TeamID != testTeamID {
		t.Fatalf("expected store call for team %s; got %#v", testTeamID, store.TopReactionsForTeamCalls)
	}
}

func TestAPI_TopReactionsForTeam_invalidTimeRange(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/reactions", testUserID))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want 400 for missing time range", w.Code)
	}
}
