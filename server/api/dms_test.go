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

func decodeDMs(t *testing.T, w *httptest.ResponseRecorder) *insights.TopDMList {
	t.Helper()
	var body insights.TopDMList
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return &body
}

func TestAPI_TopDMsForUser_unauthenticated(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	api := New(&apitest.AuthStub{}, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/me/top/dms?time_range=today", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", w.Code)
	}
}

func TestAPI_TopDMsForUser_guest(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	auth := &apitest.AuthStub{Users: map[string]*model.User{testGuestID: newGuestUser(testGuestID)}}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/dms?time_range=today", testGuestID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403 for guest", w.Code)
	}
}

func TestAPI_TopDMsForUser_dividesMessageCountByTwoAndHydrates(t *testing.T) {
	skipIfPersonalInsightsDisabled(t)
	const partnerID = "partner1aaaaaaaaaaaaaaaaaa"

	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	directory := &apitest.DirectoryStub{
		Users: map[string]*model.User{
			partnerID: {
				Id: partnerID, Username: "partner.one",
				FirstName: "Partner", LastName: "One",
				Position: "Director", Nickname: "P1",
			},
		},
	}
	store := &apitest.StoreStub{
		TopDMsForUserResult: &insights.TopDMList{
			Items: []*insights.TopDM{
				{ChannelID: "dm1", MessageCount: 20, Participants: testUserID + "," + partnerID},
			},
		},
		OutgoingDMCountsResult: map[string]int64{"dm1": 7},
	}
	api := New(auth, directory, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/dms?time_range=today", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeDMs(t, w)
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 DM; got %#v", body.Items)
	}
	dm := body.Items[0]
	// store returned 20 (raw 2x); handler must divide by 2 → 10.
	if dm.MessageCount != 10 {
		t.Errorf("MessageCount = %d; want 10 (raw 20/2)", dm.MessageCount)
	}
	if dm.OutgoingMessageCount != 7 {
		t.Errorf("OutgoingMessageCount = %d; want 7", dm.OutgoingMessageCount)
	}
	if dm.SecondParticipant == nil || dm.SecondParticipant.Username != "partner.one" {
		t.Errorf("SecondParticipant = %#v; want partner.one", dm.SecondParticipant)
	}
	if dm.SecondParticipant.Position != "Director" {
		t.Errorf("Position = %q; want Director", dm.SecondParticipant.Position)
	}

	// Ensure the outgoing query received the right channel ids.
	if len(store.OutgoingDMCountsCalls) != 1 || len(store.OutgoingDMCountsCalls[0].ChannelIDs) != 1 || store.OutgoingDMCountsCalls[0].ChannelIDs[0] != "dm1" {
		t.Errorf("OutgoingDMCounts call = %#v", store.OutgoingDMCountsCalls)
	}
}
