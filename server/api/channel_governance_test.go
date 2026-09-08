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

func governanceAPI() (*API, *apitest.StoreStub) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	store := &apitest.StoreStub{
		ChannelActivityResult: []*insights.ChannelActivity{
			{
				ID: "ch0aaaaaaaaaaaaaaaaaaaaaaa", Name: "busy", Type: model.ChannelTypeOpen,
				Purpose: "standup", Header: "links here",
				MessageCount: 100, ActivePosters: 12, MemberCount: 30,
				CreateAt: 1, LastPostAt: 900,
			},
			{
				ID: "ch1aaaaaaaaaaaaaaaaaaaaaaa", Name: "abandoned", Type: model.ChannelTypeOpen,
				Purpose: "", Header: "",
				MessageCount: 0, ActivePosters: 0, MemberCount: 80,
				CreateAt: 1, LastPostAt: 5,
			},
			{
				ID: "ch2aaaaaaaaaaaaaaaaaaaaaaa", Name: "quiet", Type: model.ChannelTypeOpen,
				Purpose: "archive of old decisions", Header: "",
				MessageCount: 0, ActivePosters: 0, MemberCount: 4,
				CreateAt: 1, LastPostAt: 0,
			},
		},
	}
	return New(auth, &apitest.DirectoryStub{}, store), store
}

func getGovernance(t *testing.T, api *API, query string) *insights.ChannelGovernanceList {
	t.Helper()
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/channel_activity?"+query, testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	var body insights.ChannelGovernanceList
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return &body
}

func TestGovernance_returnsEveryChannelWithMetadata(t *testing.T) {
	api, _ := governanceAPI()
	body := getGovernance(t, api, "time_range=7_day")

	if len(body.Items) != 3 {
		t.Fatalf("got %d channels; want all 3 — the long tail is the point", len(body.Items))
	}
	top := body.Items[0]
	if top.Name != "busy" {
		t.Errorf("first row = %s; want busy (highest message count)", top.Name)
	}
	if top.ActivePosters != 12 || top.MemberCount != 30 || top.Purpose != "standup" {
		t.Errorf("metadata not carried through: %+v", top)
	}
}

// The signal the customer actually asked for: a channel with many members and
// no posts has to survive to the response rather than being filtered as empty.
func TestGovernance_keepsChannelsWithNoActivity(t *testing.T) {
	api, _ := governanceAPI()
	body := getGovernance(t, api, "time_range=7_day")

	var found *insights.ChannelActivity
	for _, c := range body.Items {
		if c.Name == "abandoned" {
			found = c
		}
	}
	if found == nil {
		t.Fatal("the zero-post channel was dropped")
	}
	if found.MemberCount != 80 || found.MessageCount != 0 {
		t.Errorf("got %+v; want 80 members and 0 posts", found)
	}
}

func TestGovernance_summarizesLabellingCoverage(t *testing.T) {
	api, _ := governanceAPI()
	body := getGovernance(t, api, "time_range=7_day")

	s := body.Summary
	if s.TotalChannels != 3 {
		t.Errorf("TotalChannels = %d; want 3", s.TotalChannels)
	}
	if s.WithPurpose != 2 {
		t.Errorf("WithPurpose = %d; want 2", s.WithPurpose)
	}
	if s.WithHeader != 1 {
		t.Errorf("WithHeader = %d; want 1", s.WithHeader)
	}
	if s.ActiveChannels != 1 {
		t.Errorf("ActiveChannels = %d; want 1", s.ActiveChannels)
	}
}

// The summary describes the team, not the page — a client holding page 2 of 40
// still needs the correct denominator.
func TestGovernance_summaryCoversAllChannelsNotJustThePage(t *testing.T) {
	api, _ := governanceAPI()
	body := getGovernance(t, api, "time_range=7_day&per_page=1&page=0")

	if len(body.Items) != 1 {
		t.Fatalf("got %d items; want 1 (per_page=1)", len(body.Items))
	}
	if body.Summary.TotalChannels != 3 {
		t.Errorf("Summary.TotalChannels = %d on a 1-item page; want 3", body.Summary.TotalChannels)
	}
	if !body.HasNext {
		t.Error("has_next should be true")
	}
}

// Governance reads the same cached aggregate as the cards, so opening the page
// must not add another aggregation.
func TestGovernance_sharesTheCachedAggregateWithTheCards(t *testing.T) {
	api, store := governanceAPI()

	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/channels?time_range=7_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("top/channels status = %d", w.Code)
	}
	getGovernance(t, api, "time_range=7_day")

	if n := len(store.ChannelActivityCalls); n != 1 {
		t.Errorf("ChannelActivityForTeam ran %d times; want 1 shared between the card and the table", n)
	}
}

func TestGovernance_respectsPrivateChannelVisibility(t *testing.T) {
	api, store := governanceAPI()
	store.ChannelActivityResult = append(store.ChannelActivityResult, &insights.ChannelActivity{
		ID: "secretaaaaaaaaaaaaaaaaaaaa", Name: "secret", Type: model.ChannelTypePrivate,
		MessageCount: 500, CreateAt: 1,
	})

	body := getGovernance(t, api, "time_range=7_day")
	for _, c := range body.Items {
		if c.Name == "secret" {
			t.Fatal("a private channel the requester is not a member of leaked into the table")
		}
	}
	if body.Summary.TotalChannels != 3 {
		t.Errorf("Summary.TotalChannels = %d; want 3 — the summary must count visible channels only", body.Summary.TotalChannels)
	}
}
