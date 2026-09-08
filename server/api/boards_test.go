package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/api/apitest"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

func newBoardsAPI() (*API, *apitest.AuthStub, *apitest.StoreStub, *apitest.DirectoryStub) {
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
		BoardIDsForUserInTeamResult: []string{"b1", "b2"},
		TopBoardsForTeamResult: &insights.TopBoardList{
			ListData: insights.ListData{HasNext: false},
			Items: []*insights.TopBoard{
				{BoardID: "b1", Title: "Board One", Icon: "💬", ActivityCount: "9", ActiveUsers: []string{"u1"}, CreatedBy: "u1"},
			},
		},
		TopBoardsForUserResult: &insights.TopBoardList{
			ListData: insights.ListData{HasNext: false},
			Items: []*insights.TopBoard{
				{BoardID: "b1", Title: "Board One", ActivityCount: "5", ActiveUsers: []string{"u1"}, CreatedBy: "u1"},
			},
		},
	}
	dir := &apitest.DirectoryStub{Users: map[string]*model.User{}}
	return New(auth, dir, store), auth, store, dir
}

func proLicense() *model.License {
	sku := model.LicenseShortSkuProfessional
	return &model.License{Features: &model.Features{}, SkuShortName: sku}
}

func TestTopBoardsForTeam_happyPath(t *testing.T) {
	skipIfBoardsAndPlaybooksDisabled(t)
	api, _, store, _ := newBoardsAPI()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/team1/top/boards?time_range=7_day", nil)
	req.Header.Set("Mattermost-User-Id", "u1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(store.BoardIDsForUserInTeamCalls) != 1 {
		t.Fatalf("expected one BoardIDsForUserInTeam call, got %v", store.BoardIDsForUserInTeamCalls)
	}
	if call := store.BoardIDsForUserInTeamCalls[0]; call.UserID != "u1" || call.TeamID != "team1" {
		t.Errorf("BoardIDsForUserInTeam called with wrong args: %+v", call)
	}
	if len(store.TopBoardsForTeamCalls) != 1 {
		t.Fatalf("expected one TopBoardsForTeam call, got %v", store.TopBoardsForTeamCalls)
	}
	c := store.TopBoardsForTeamCalls[0]
	if c.TeamID != "team1" || len(c.BoardIDs) != 2 || c.BoardIDs[0] != "b1" {
		t.Errorf("TopBoardsForTeam called with wrong args: %+v", c)
	}
}

func TestTopBoardsForTeam_rejectsWithoutLicense(t *testing.T) {
	api, auth, _, _ := newBoardsAPI()
	auth.License = nil

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/team1/top/boards?time_range=7_day", nil)
	req.Header.Set("Mattermost-User-Id", "u1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without license, got %d", rec.Code)
	}
}

func TestTopBoardsForUser_happyPath(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	api, _, store, _ := newBoardsAPI()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/top/boards?time_range=7_day&team_id=team1", nil)
	req.Header.Set("Mattermost-User-Id", "u1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(store.TopBoardsForUserCalls) != 1 {
		t.Fatalf("expected one TopBoardsForUser call, got %v", store.TopBoardsForUserCalls)
	}
	c := store.TopBoardsForUserCalls[0]
	if c.TeamID != "team1" || c.UserID != "u1" || len(c.BoardIDs) != 2 {
		t.Errorf("TopBoardsForUser called with wrong args: %+v", c)
	}
}

func TestTopBoardsForUser_rejectsMissingTeamId(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	api, _, _, _ := newBoardsAPI()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/top/boards?time_range=7_day", nil)
	req.Header.Set("Mattermost-User-Id", "u1")
	rec := httptest.NewRecorder()
	api.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without team_id, got %d", rec.Code)
	}
}
