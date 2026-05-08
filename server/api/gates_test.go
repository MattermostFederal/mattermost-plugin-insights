package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/api/apitest"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// Gates matrix — expressed once, applied to every route. The per-insight
// tests already cover happy paths; this file enforces that the gate behavior
// is applied uniformly across the API surface so a future handler can't
// silently skip a check. Adapted from the deprecated
// server/channels/api4/insights_test.go's auth sub-tests.

// allRoutes returns the (path, scope) pairs the router serves for the
// gates-matrix tests. scope is "team" if the route is team-scoped (and
// therefore subject to the license + view_team gates), "user" otherwise.
func allRoutes() []struct {
	path, scope string
} {
	return []struct{ path, scope string }{
		{"/api/v1/teams/" + testTeamID + "/top/reactions?time_range=today", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/channels?time_range=today", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/threads?time_range=today", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/inactive_channels?time_range=today", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/team_members?time_range=today", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/boards?time_range=today", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/playbooks?time_range=today", "team"},
		{"/api/v1/users/me/top/reactions?time_range=today", "user"},
		{"/api/v1/users/me/top/channels?time_range=today", "user"},
		{"/api/v1/users/me/top/threads?time_range=today", "user"},
		{"/api/v1/users/me/top/dms?time_range=today", "user"},
		{"/api/v1/users/me/top/inactive_channels?time_range=today", "user"},
		{"/api/v1/users/me/top/boards?time_range=today&team_id=" + testTeamID, "user"},
		{"/api/v1/users/me/top/playbooks?time_range=today&team_id=" + testTeamID, "user"},
	}
}

func TestGates_unauthenticatedAlways401(t *testing.T) {
	api := New(&apitest.AuthStub{}, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	for _, route := range allRoutes() {
		t.Run(route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			api.ServeHTTP(w, httptest.NewRequest(http.MethodGet, route.path, nil))
			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d; want 401", w.Code)
			}
		})
	}
}

func TestGates_guestAlways403(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testGuestID: newGuestUser(testGuestID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testGuestID: {testTeamID: true}},
	}
	api := New(auth, &apitest.DirectoryStub{}, fullyStubbedStore())
	for _, route := range allRoutes() {
		t.Run(route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			api.ServeHTTP(w, newAuthedRequest(http.MethodGet, route.path, testGuestID))
			if w.Code != http.StatusForbidden {
				t.Errorf("status = %d; want 403 (guest)", w.Code)
			}
		})
	}
}

func TestGates_teamRoutesRequireProfessionalLicense(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
		// no License -> license check should fail.
	}
	api := New(auth, &apitest.DirectoryStub{}, fullyStubbedStore())
	for _, route := range allRoutes() {
		t.Run(route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			api.ServeHTTP(w, newAuthedRequest(http.MethodGet, route.path, testUserID))
			if route.scope == "team" {
				if w.Code != http.StatusForbidden {
					t.Errorf("team route w/o license: status = %d; want 403", w.Code)
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("user route w/o license: status = %d; want 200; body=%s", w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestGates_teamRoutesRequireTeamPermission(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:   map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License: professionalLicense(),
		// no TeamPerms entry -> view_team check should fail.
	}
	api := New(auth, &apitest.DirectoryStub{}, fullyStubbedStore())
	for _, route := range allRoutes() {
		t.Run(route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			api.ServeHTTP(w, newAuthedRequest(http.MethodGet, route.path, testUserID))
			if route.scope == "team" {
				if w.Code != http.StatusForbidden {
					t.Errorf("team route w/o team perm: status = %d; want 403", w.Code)
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("user route ignores team perm: status = %d; want 200; body=%s", w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestGates_authorizedRequestsSucceed(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	api := New(auth, &apitest.DirectoryStub{}, fullyStubbedStore())
	for _, route := range allRoutes() {
		t.Run(route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			api.ServeHTTP(w, newAuthedRequest(http.MethodGet, route.path, testUserID))
			if w.Code != http.StatusOK {
				t.Errorf("status = %d; want 200; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

// TestGates_acceptsAllSupportedLicenseTiers verifies the team-scoped routes
// pass the license gate for every Mattermost license tier at Professional or
// above — Professional, Enterprise, and Enterprise Advanced. Encoded as a
// regression test so a future tightening of the license check (or a model
// package change) can't silently lock out higher tiers.
func TestGates_acceptsAllSupportedLicenseTiers(t *testing.T) {
	tiers := []struct {
		name string
		sku  string
	}{
		{"Professional", model.LicenseShortSkuProfessional},
		{"Enterprise", model.LicenseShortSkuEnterprise},
		{"Enterprise Advanced", model.LicenseShortSkuEnterpriseAdvanced},
	}
	for _, tier := range tiers {
		t.Run(tier.name, func(t *testing.T) {
			auth := &apitest.AuthStub{
				Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
				License:   model.NewTestLicenseSKU(tier.sku),
				TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
			}
			api := New(auth, &apitest.DirectoryStub{}, fullyStubbedStore())
			w := httptest.NewRecorder()
			api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/reactions?time_range=today", testUserID))
			if w.Code != http.StatusOK {
				t.Errorf("%s license: status = %d; want 200; body=%s", tier.name, w.Code, w.Body.String())
			}
		})
	}
}

// fullyStubbedStore returns a StoreStub with empty results for every method
// so the gates matrix exercises auth without tripping on missing data.
func fullyStubbedStore() *apitest.StoreStub {
	return &apitest.StoreStub{
		TopReactionsForUserResult:        &insights.TopReactionList{},
		TopReactionsForTeamResult:        &insights.TopReactionList{},
		TopThreadsForUserResult:          &insights.TopThreadList{},
		TopThreadsForTeamResult:          &insights.TopThreadList{},
		NewTeamMembersResult:             &insights.NewTeamMembersList{},
		TopChannelsForUserResult:         &insights.TopChannelList{},
		TopChannelsForTeamResult:         &insights.TopChannelList{},
		TopInactiveChannelsForUserResult: &insights.TopInactiveChannelList{},
		TopInactiveChannelsForTeamResult: &insights.TopInactiveChannelList{},
		TopDMsForUserResult:              &insights.TopDMList{},
	}
}
