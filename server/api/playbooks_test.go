package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/api/apitest"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

func newPlaybooksAPI() (*API, *apitest.AuthStub, *apitest.StoreStub) {
	auth := &apitest.AuthStub{
		Users: map[string]*model.User{
			"u1": {Id: "u1"},
		},
		License: proLicense(),
		TeamPerms: map[string]map[string]bool{
			"u1": {"team1": true},
		},
	}
	store := &apitest.StoreStub{
		TopPlaybooksForTeamResult: &insights.TopPlaybookList{
			ListData: insights.ListData{HasNext: false},
			Items: []*insights.TopPlaybook{
				{PlaybookID: "pb1", Title: "Incident response", NumRuns: 12, LastRunAt: 1_700_000_000_000},
			},
		},
		TopPlaybooksForUserResult: &insights.TopPlaybookList{
			ListData: insights.ListData{HasNext: false},
			Items: []*insights.TopPlaybook{
				{PlaybookID: "pb2", Title: "Onboarding", NumRuns: 4, LastRunAt: 1_700_000_000_000},
			},
		},
	}
	return New(auth, &apitest.DirectoryStub{}, store), auth, store
}

func TestTopPlaybooksForTeam_happyPath(t *testing.T) {
	skipIfBoardsAndPlaybooksDisabled(t)
	api, _, store := newPlaybooksAPI()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/team1/top/playbooks?time_range=7_day", nil)
	req.Header.Set("Mattermost-User-Id", "u1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(store.TopPlaybooksForTeamCalls) != 1 {
		t.Fatalf("expected one TopPlaybooksForTeam call, got %v", store.TopPlaybooksForTeamCalls)
	}
	c := store.TopPlaybooksForTeamCalls[0]
	if c.TeamID != "team1" || c.UserID != "u1" {
		t.Errorf("call args mismatch: %+v", c)
	}
}

// Regression: the customer's Enterprise Advanced license must NOT be
// rejected by the team-scope handler. The Playbooks plugin's
// `licenseAndGuestCheck` rejects `advanced` SKU with a 500; this plugin
// uses `model.MinimumProfessionalLicense` which accepts it.
// Note: this deliberately asserts only the status code. The 200-vs-403
// distinction is the whole regression — whether the store is reached is
// incidental, and is short-circuited while EnableBoardsAndPlaybooks is off.
func TestTopPlaybooksForTeam_acceptsEnterpriseAdvancedLicense(t *testing.T) {
	api, auth, _ := newPlaybooksAPI()
	auth.License = &model.License{
		Features:     &model.Features{},
		SkuShortName: model.LicenseShortSkuEnterpriseAdvanced,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/team1/top/playbooks?time_range=7_day", nil)
	req.Header.Set("Mattermost-User-Id", "u1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Enterprise Advanced license should be accepted; got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTopPlaybooksForTeam_rejectsWithoutLicense(t *testing.T) {
	api, auth, _ := newPlaybooksAPI()
	auth.License = nil

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/team1/top/playbooks?time_range=7_day", nil)
	req.Header.Set("Mattermost-User-Id", "u1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without license, got %d", rec.Code)
	}
}

func TestTopPlaybooksForUser_happyPath(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	api, _, store := newPlaybooksAPI()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/top/playbooks?time_range=7_day&team_id=team1", nil)
	req.Header.Set("Mattermost-User-Id", "u1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(store.TopPlaybooksForUserCalls) != 1 {
		t.Fatalf("expected one TopPlaybooksForUser call, got %v", store.TopPlaybooksForUserCalls)
	}
	c := store.TopPlaybooksForUserCalls[0]
	if c.TeamID != "team1" || c.UserID != "u1" {
		t.Errorf("call args mismatch: %+v", c)
	}
}

func TestTopPlaybooksForUser_rejectsMissingTeamId(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	api, _, _ := newPlaybooksAPI()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/top/playbooks?time_range=7_day", nil)
	req.Header.Set("Mattermost-User-Id", "u1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without team_id, got %d", rec.Code)
	}
}
