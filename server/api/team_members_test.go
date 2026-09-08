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

func decodeNewTeamMembers(t *testing.T, w *httptest.ResponseRecorder) *insights.NewTeamMembersList {
	t.Helper()
	var body insights.NewTeamMembersList
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return &body
}

func newAdminUser(id string) *model.User {
	return &model.User{Id: id, Roles: "system_user system_admin"}
}

func TestAPI_NewTeamMembers_unauthenticated(t *testing.T) {
	api := New(&apitest.AuthStub{}, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/team_members?time_range=7_day", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", w.Code)
	}
}

func TestAPI_NewTeamMembers_noLicense(t *testing.T) {
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/team_members?time_range=7_day", testUserID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403 without license", w.Code)
	}
}

func TestAPI_NewTeamMembers_notMember(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:   map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License: professionalLicense(),
	}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/team_members?time_range=7_day", testUserID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403 when not a team member", w.Code)
	}
}

func TestAPI_NewTeamMembers_returnsItemsAndTotalCount(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	store := &apitest.StoreStub{
		NewTeamMembersResult: &insights.NewTeamMembersList{
			ListData: insights.ListData{HasNext: true},
			Items: []*insights.NewTeamMember{
				{ID: "u1", Username: "alice", FirstName: "Alice", LastName: "Anders"},
			},
			TotalCount: 12,
		},
	}
	api := New(auth, &apitest.DirectoryStub{}, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/team_members?time_range=7_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeNewTeamMembers(t, w)
	if body.TotalCount != 12 || !body.HasNext || len(body.Items) != 1 || body.Items[0].Username != "alice" {
		t.Fatalf("body = %#v", body)
	}
	if len(store.NewTeamMembersCalls) != 1 {
		t.Fatalf("expected 1 store call; got %d", len(store.NewTeamMembersCalls))
	}
	if !store.NewTeamMembersCalls[0].ShowFullName {
		t.Errorf("expected ShowFullName=true (default privacy); got false")
	}
}

func TestAPI_NewTeamMembers_passesShowFullNameForAdminEvenWhenPrivacyOff(t *testing.T) {
	adminID := "admin1aaaaaaaaaaaaaaaaaaaa"
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{adminID: newAdminUser(adminID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{adminID: {testTeamID: true}},
	}
	directory := &apitest.DirectoryStub{ShowFullNameSetting: new(bool)}
	store := &apitest.StoreStub{NewTeamMembersResult: &insights.NewTeamMembersList{}}
	api := New(auth, directory, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/team_members?time_range=7_day", adminID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	if !store.NewTeamMembersCalls[0].ShowFullName {
		t.Errorf("admin should always get ShowFullName=true; got false")
	}
}

func TestAPI_NewTeamMembers_passesShowFullNameFalseForNonAdminWhenPrivacyOff(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	directory := &apitest.DirectoryStub{ShowFullNameSetting: new(bool)}
	store := &apitest.StoreStub{NewTeamMembersResult: &insights.NewTeamMembersList{}}
	api := New(auth, directory, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/team_members?time_range=7_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	if store.NewTeamMembersCalls[0].ShowFullName {
		t.Errorf("non-admin with privacy=false should get ShowFullName=false")
	}
}
