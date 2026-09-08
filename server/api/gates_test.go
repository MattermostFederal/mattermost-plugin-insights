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

// Gates matrix — expressed once, applied to every route. The per-insight
// tests already cover happy paths; this file enforces that the gate behavior
// is applied uniformly across the API surface so a future handler can't
// silently skip a check. Adapted from the deprecated
// server/channels/api4/insights_test.go's auth sub-tests.

// skipIfPersonalInsightsDisabled marks a My-scope handler test as skipped
// while EnablePersonalInsights is off. The handlers themselves remain in the
// tree, so these tests stay valid and run again the moment the flag flips.
func skipIfPersonalInsightsDisabled(t *testing.T) {
	t.Helper()
	if !EnablePersonalInsights {
		t.Skip("personal insights disabled; see EnablePersonalInsights")
	}
}

// skipIfBoardsAndPlaybooksDisabled marks a test that asserts Top Boards /
// Top Playbooks actually query the store. While EnableBoardsAndPlaybooks is
// off the handlers short-circuit to an empty NotAvailable list, so these
// assertions only apply once the flag flips back on.
func skipIfBoardsAndPlaybooksDisabled(t *testing.T) {
	t.Helper()
	if !EnableBoardsAndPlaybooks {
		t.Skip("boards/playbooks disabled; see EnableBoardsAndPlaybooks")
	}
}

// TestBoardsAndPlaybooksStubbed pins the stub contract: the routes still
// pass their auth gates and return 200, but they query nothing and flag the
// payload NotAvailable so the webapp can tell "off" from "empty".
func TestBoardsAndPlaybooksStubbed(t *testing.T) {
	if EnableBoardsAndPlaybooks {
		t.Skip("boards/playbooks are enabled")
	}
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: {Id: testUserID}},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	for _, path := range []string{
		"/api/v1/teams/" + testTeamID + "/top/boards?time_range=7_day",
		"/api/v1/teams/" + testTeamID + "/top/playbooks?time_range=7_day",
	} {
		t.Run(path, func(t *testing.T) {
			store := fullyStubbedStore()
			api := New(auth, &apitest.DirectoryStub{}, store)

			w := httptest.NewRecorder()
			api.ServeHTTP(w, newAuthedRequest(http.MethodGet, path, testUserID))

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d; want 200, body=%s", w.Code, w.Body.String())
			}
			var body struct {
				Items        []json.RawMessage `json:"items"`
				NotAvailable bool              `json:"not_available"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !body.NotAvailable {
				t.Error("not_available = false; want true")
			}
			if len(body.Items) != 0 {
				t.Errorf("items = %d; want 0", len(body.Items))
			}
			if n := len(store.BoardIDsForUserInTeamCalls) + len(store.TopPlaybooksForTeamCalls); n != 0 {
				t.Errorf("store was queried %d times; want 0", n)
			}
		})
	}
}

// gateRoute is one row of the gates matrix. scope is "team" if the route is
// team-scoped (and therefore subject to the license + view_team gates),
// "user" otherwise.
type gateRoute struct {
	path, scope string
}

// personalRoutes are the user-scoped ("My") insight routes. They are only
// served when EnablePersonalInsights is on; TestGates_personalInsightsDisabled
// asserts they are unreachable otherwise.
func personalRoutes() []gateRoute {
	return []gateRoute{
		{"/api/v1/users/me/top/reactions?time_range=7_day", "user"},
		{"/api/v1/users/me/top/channels?time_range=7_day", "user"},
		{"/api/v1/users/me/top/threads?time_range=7_day", "user"},
		{"/api/v1/users/me/top/dms?time_range=7_day", "user"},
		{"/api/v1/users/me/top/inactive_channels?time_range=7_day", "user"},
		{"/api/v1/users/me/top/boards?time_range=7_day&team_id=" + testTeamID, "user"},
		{"/api/v1/users/me/top/playbooks?time_range=7_day&team_id=" + testTeamID, "user"},
	}
}

// allRoutes returns every route the router currently serves, for the
// gates-matrix tests.
func allRoutes() []gateRoute {
	routes := []gateRoute{
		{"/api/v1/teams/" + testTeamID + "/top/reactions?time_range=7_day", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/channels?time_range=7_day", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/threads?time_range=7_day", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/inactive_channels?time_range=7_day", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/team_members?time_range=7_day", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/boards?time_range=7_day", "team"},
		{"/api/v1/teams/" + testTeamID + "/top/playbooks?time_range=7_day", "team"},
		{"/api/v1/teams/" + testTeamID + "/channel_activity?time_range=7_day", "team"},
	}
	if EnablePersonalInsights {
		routes = append(routes, personalRoutes()...)
	}
	return routes
}

// TestGates_personalInsightsDisabled pins the Phase 1a behavior: with
// EnablePersonalInsights off, an authenticated non-guest user cannot reach
// any My-scope route, so none of their per-request aggregations can run.
func TestGates_personalInsightsDisabled(t *testing.T) {
	if EnablePersonalInsights {
		t.Skip("personal insights are enabled")
	}
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: {Id: testUserID}},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	api := New(auth, &apitest.DirectoryStub{}, fullyStubbedStore())
	for _, route := range personalRoutes() {
		t.Run(route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			api.ServeHTTP(w, newAuthedRequest(http.MethodGet, route.path, testUserID))
			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d; want 404 (personal insights disabled)", w.Code)
			}
		})
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
			api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/reactions?time_range=7_day", testUserID))
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
