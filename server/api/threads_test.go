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

func decodeThreads(t *testing.T, w *httptest.ResponseRecorder) *insights.TopThreadList {
	t.Helper()
	var body insights.TopThreadList
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return &body
}

const (
	testThreadPostID = "post0aaaaaaaaaaaaaaaaaaaaa"
	testRootAuthorID = "author0aaaaaaaaaaaaaaaaaaa"
)

func TestAPI_TopThreadsForUser_unauthenticated(t *testing.T) {
	api := New(&apitest.AuthStub{}, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/users/me/top/threads?time_range=today", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want 401", w.Code)
	}
}

func TestAPI_TopThreadsForUser_guest(t *testing.T) {
	auth := &apitest.AuthStub{Users: map[string]*model.User{testGuestID: newGuestUser(testGuestID)}}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/threads?time_range=today", testGuestID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403 for guest", w.Code)
	}
}

func TestAPI_TopThreadsForUser_hydratesUserAndPost(t *testing.T) {
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	directory := &apitest.DirectoryStub{
		Users: map[string]*model.User{
			testRootAuthorID: {
				Id: testRootAuthorID, Username: "thread.author",
				FirstName: "Thread", LastName: "Author",
				Nickname: "ta", LastPictureUpdate: 12345,
			},
		},
		Posts: map[string]*model.Post{
			testThreadPostID: {Id: testThreadPostID, Message: "root post"},
		},
	}
	store := &apitest.StoreStub{
		TopThreadsForUserResult: &insights.TopThreadList{
			Items: []*insights.TopThread{
				{
					PostID:     testThreadPostID,
					ReplyCount: 7,
					ChannelID:  "ch1aaaaaaaaaaaaaaaaaaaaaaa",
					UserID:     testRootAuthorID,
				},
			},
		},
	}
	api := New(auth, directory, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/users/me/top/threads?time_range=7_day", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeThreads(t, w)
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 item; got %#v", body.Items)
	}
	item := body.Items[0]
	if item.UserInformation == nil {
		t.Fatalf("expected hydrated UserInformation; got nil")
	}
	if item.UserInformation.Username != "thread.author" || item.UserInformation.FirstName != "Thread" {
		t.Errorf("user info = %#v", item.UserInformation)
	}
	if item.Post == nil || item.Post.Id != testThreadPostID {
		t.Errorf("expected hydrated Post with id %s; got %#v", testThreadPostID, item.Post)
	}
	if len(store.TopThreadsForUserCalls) != 1 {
		t.Fatalf("expected 1 store call; got %d", len(store.TopThreadsForUserCalls))
	}
}

func TestAPI_TopThreadsForTeam_noLicense(t *testing.T) {
	auth := &apitest.AuthStub{Users: map[string]*model.User{testUserID: newRegularUser(testUserID)}}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/threads?time_range=today", testUserID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403 without Professional license", w.Code)
	}
}

func TestAPI_TopThreadsForTeam_notMember(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:   map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License: professionalLicense(),
	}
	api := New(auth, &apitest.DirectoryStub{}, &apitest.StoreStub{})
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/threads?time_range=today", testUserID))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; want 403 when not a team member", w.Code)
	}
}

func TestAPI_TopThreadsForTeam_returnsHydratedItems(t *testing.T) {
	auth := &apitest.AuthStub{
		Users:     map[string]*model.User{testUserID: newRegularUser(testUserID)},
		License:   professionalLicense(),
		TeamPerms: map[string]map[string]bool{testUserID: {testTeamID: true}},
	}
	directory := &apitest.DirectoryStub{
		Users: map[string]*model.User{
			testRootAuthorID: {Id: testRootAuthorID, Username: "thread.author"},
		},
		Posts: map[string]*model.Post{
			testThreadPostID: {Id: testThreadPostID, Message: "root post"},
		},
	}
	store := &apitest.StoreStub{
		TopThreadsForTeamResult: &insights.TopThreadList{
			Items: []*insights.TopThread{
				{PostID: testThreadPostID, ReplyCount: 4, ChannelID: "ch", UserID: testRootAuthorID},
			},
		},
	}
	api := New(auth, directory, store)
	w := httptest.NewRecorder()
	api.ServeHTTP(w, newAuthedRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/top/threads?time_range=today", testUserID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200; body=%s", w.Code, w.Body.String())
	}
	body := decodeThreads(t, w)
	if len(body.Items) != 1 || body.Items[0].UserInformation == nil || body.Items[0].UserInformation.Username != "thread.author" {
		t.Fatalf("body = %#v", body)
	}
	if body.Items[0].Post == nil || body.Items[0].Post.Id != testThreadPostID {
		t.Fatalf("expected hydrated post; got %#v", body.Items[0].Post)
	}
}
