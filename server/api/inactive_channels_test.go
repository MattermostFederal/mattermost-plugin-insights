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

func decodeInactiveChannels(t *testing.T, w *httptest.ResponseRecorder) *insights.TopInactiveChannelList {
	t.Helper()
	var body insights.TopInactiveChannelList
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return &body
}

func TestAPI_TopInactiveChannelsForUser_unauthenticated(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	api := New(&apitest.AuthStub{}, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/me/top/inactive_channels?time_range=today", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", w.Code)
	}
}

func TestAPI_TopInactiveChannelsForUser_returnsItems(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	store := &apitest.StoreStub{
		TopInactiveChannelsForUserResult: &insights.TopInactiveChannelList{
			Items: []*insights.TopInactiveChannel{
				{ID: "ch1", Name: "quiet", DisplayName: "Quiet", MessageCount: 0},
			},
		},
	}
	api := New(auth, &apitest.DirectoryStub{}, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/inactive_channels?time_range=28_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeInactiveChannels(t, w)
	if len(body.Items) != 1 || body.Items[0].Name != "quiet" {
		t.Fatalf("body = %#v", body)
	}
}

func TestAPI_TopInactiveChannelsForTeam_noLicense(t *testing.T) {
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/inactive_channels?time_range=today", testUserID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403", w.Code)
	}
}

func TestAPI_TopInactiveChannelsForTeam_returnsItems(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	store := &apitest.StoreStub{
		TopInactiveChannelsForTeamResult: &insights.TopInactiveChannelList{
			Items: []*insights.TopInactiveChannel{{ID: "ch", Name: "deserted", MessageCount: 0, Participants: model.StringArray{"u1"}}},
		},
	}
	api := New(auth, &apitest.DirectoryStub{}, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/inactive_channels?time_range=today", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeInactiveChannels(t, w)
	if len(body.Items) != 1 || body.Items[0].Name != "deserted" || len(body.Items[0].Participants) != 1 {
		t.Fatalf("body = %#v", body)
	}
}
