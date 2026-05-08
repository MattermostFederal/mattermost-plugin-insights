// Package apitest provides stubs that satisfy the api.AuthProvider and
// api.Storer interfaces, used by handler-level tests in the api package.
package apitest

import (
	"context"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/insights"
)

// AuthStub is an in-memory implementation of api.AuthProvider for tests.
type AuthStub struct {
	Users      map[string]*model.User
	License    *model.License
	TeamPerms  map[string]map[string]bool // userID -> teamID -> allowed
	GetUserErr error
}

func (s *AuthStub) GetUser(userID string) (*model.User, error) {
	if s.GetUserErr != nil {
		return nil, s.GetUserErr
	}
	if u, ok := s.Users[userID]; ok {
		return u, nil
	}
	return nil, &model.AppError{Message: "user not found", StatusCode: 404}
}

func (s *AuthStub) HasPermissionToTeam(userID, teamID string, _ *model.Permission) bool {
	if perms, ok := s.TeamPerms[userID]; ok {
		return perms[teamID]
	}
	return false
}

func (s *AuthStub) GetLicense() *model.License {
	return s.License
}

// StoreStub is an in-memory implementation of api.Storer for tests. Each
// field captures the most recent call's arguments so tests can assert.
type StoreStub struct {
	TopReactionsForUserResult *insights.TopReactionList
	TopReactionsForUserErr    error
	TopReactionsForUserCalls  []ReactionsForUserCall

	TopReactionsForTeamResult *insights.TopReactionList
	TopReactionsForTeamErr    error
	TopReactionsForTeamCalls  []ReactionsForTeamCall

	TopThreadsForUserResult *insights.TopThreadList
	TopThreadsForUserErr    error
	TopThreadsForUserCalls  []ThreadsForUserCall

	TopThreadsForTeamResult *insights.TopThreadList
	TopThreadsForTeamErr    error
	TopThreadsForTeamCalls  []ThreadsForTeamCall

	NewTeamMembersResult *insights.NewTeamMembersList
	NewTeamMembersErr    error
	NewTeamMembersCalls  []NewTeamMembersCall

	TopChannelsForUserResult *insights.TopChannelList
	TopChannelsForUserErr    error
	TopChannelsForUserCalls  []ChannelsForUserCall

	TopChannelsForTeamResult *insights.TopChannelList
	TopChannelsForTeamErr    error
	TopChannelsForTeamCalls  []ChannelsForTeamCall

	TopInactiveChannelsForUserResult *insights.TopInactiveChannelList
	TopInactiveChannelsForUserErr    error
	TopInactiveChannelsForUserCalls  []InactiveChannelsForUserCall

	TopInactiveChannelsForTeamResult *insights.TopInactiveChannelList
	TopInactiveChannelsForTeamErr    error
	TopInactiveChannelsForTeamCalls  []InactiveChannelsForTeamCall

	TopDMsForUserResult *insights.TopDMList
	TopDMsForUserErr    error
	TopDMsForUserCalls  []DMsForUserCall

	OutgoingDMCountsResult map[string]int64
	OutgoingDMCountsErr    error
	OutgoingDMCountsCalls  []OutgoingDMCountsCall

	PostCountsByDurationResult []*insights.DurationPostCount
	PostCountsByDurationErr    error
	PostCountsByDurationCalls  []PostCountsByDurationCall

	BoardIDsForUserInTeamResult []string
	BoardIDsForUserInTeamErr    error
	BoardIDsForUserInTeamCalls  []BoardIDsForUserInTeamCall

	TopBoardsForTeamResult *insights.TopBoardList
	TopBoardsForTeamErr    error
	TopBoardsForTeamCalls  []BoardsForTeamCall

	TopBoardsForUserResult *insights.TopBoardList
	TopBoardsForUserErr    error
	TopBoardsForUserCalls  []BoardsForUserCall

	TopPlaybooksForTeamResult *insights.TopPlaybookList
	TopPlaybooksForTeamErr    error
	TopPlaybooksForTeamCalls  []PlaybooksForTeamCall

	TopPlaybooksForUserResult *insights.TopPlaybookList
	TopPlaybooksForUserErr    error
	TopPlaybooksForUserCalls  []PlaybooksForUserCall
}

type PlaybooksForTeamCall struct {
	TeamID, UserID string
	Since          int64
	Page, PerPage  int
}

type PlaybooksForUserCall struct {
	TeamID, UserID string
	Since          int64
	Page, PerPage  int
}

type BoardIDsForUserInTeamCall struct {
	UserID, TeamID string
}

type BoardsForTeamCall struct {
	TeamID        string
	BoardIDs      []string
	Since         int64
	Page, PerPage int
}

type BoardsForUserCall struct {
	TeamID, UserID string
	BoardIDs       []string
	Since          int64
	Page, PerPage  int
}

type PostCountsByDurationCall struct {
	ChannelIDs       []string
	SinceUnixMillis  int64
	UserID, Grouping string
	Location         string
}

type DMsForUserCall struct {
	UserID        string
	Since         int64
	Page, PerPage int
}

type OutgoingDMCountsCall struct {
	UserID     string
	ChannelIDs []string
	Since      int64
}

type InactiveChannelsForUserCall struct {
	UserID, TeamID string
	Since          int64
	Page, PerPage  int
}

type InactiveChannelsForTeamCall struct {
	TeamID, UserID string
	Since          int64
	Page, PerPage  int
}

type ChannelsForUserCall struct {
	UserID, TeamID string
	Since          int64
	Page, PerPage  int
}

type ChannelsForTeamCall struct {
	TeamID, UserID string
	Since          int64
	Page, PerPage  int
}

type NewTeamMembersCall struct {
	TeamID        string
	Since         int64
	Page, PerPage int
	ShowFullName  bool
}

type ReactionsForUserCall struct {
	UserID, TeamID string
	Since          int64
	Page, PerPage  int
}

type ReactionsForTeamCall struct {
	TeamID, UserID string
	Since          int64
	Page, PerPage  int
}

type ThreadsForUserCall struct {
	UserID, TeamID string
	Since          int64
	Page, PerPage  int
}

type ThreadsForTeamCall struct {
	TeamID, UserID string
	Since          int64
	Page, PerPage  int
}

func (s *StoreStub) TopReactionsForUserSince(_ context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopReactionList, error) {
	s.TopReactionsForUserCalls = append(s.TopReactionsForUserCalls, ReactionsForUserCall{userID, teamID, since, page, perPage})
	return s.TopReactionsForUserResult, s.TopReactionsForUserErr
}

func (s *StoreStub) TopReactionsForTeamSince(_ context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopReactionList, error) {
	s.TopReactionsForTeamCalls = append(s.TopReactionsForTeamCalls, ReactionsForTeamCall{teamID, userID, since, page, perPage})
	return s.TopReactionsForTeamResult, s.TopReactionsForTeamErr
}

func (s *StoreStub) TopThreadsForUserSince(_ context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopThreadList, error) {
	s.TopThreadsForUserCalls = append(s.TopThreadsForUserCalls, ThreadsForUserCall{userID, teamID, since, page, perPage})
	return s.TopThreadsForUserResult, s.TopThreadsForUserErr
}

func (s *StoreStub) TopThreadsForTeamSince(_ context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopThreadList, error) {
	s.TopThreadsForTeamCalls = append(s.TopThreadsForTeamCalls, ThreadsForTeamCall{teamID, userID, since, page, perPage})
	return s.TopThreadsForTeamResult, s.TopThreadsForTeamErr
}

func (s *StoreStub) NewTeamMembersSince(_ context.Context, teamID string, since int64, page, perPage int, showFullName bool) (*insights.NewTeamMembersList, error) {
	s.NewTeamMembersCalls = append(s.NewTeamMembersCalls, NewTeamMembersCall{teamID, since, page, perPage, showFullName})
	return s.NewTeamMembersResult, s.NewTeamMembersErr
}

func (s *StoreStub) TopChannelsForUserSince(_ context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopChannelList, error) {
	s.TopChannelsForUserCalls = append(s.TopChannelsForUserCalls, ChannelsForUserCall{userID, teamID, since, page, perPage})
	return s.TopChannelsForUserResult, s.TopChannelsForUserErr
}

func (s *StoreStub) TopChannelsForTeamSince(_ context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopChannelList, error) {
	s.TopChannelsForTeamCalls = append(s.TopChannelsForTeamCalls, ChannelsForTeamCall{teamID, userID, since, page, perPage})
	return s.TopChannelsForTeamResult, s.TopChannelsForTeamErr
}

func (s *StoreStub) TopInactiveChannelsForUserSince(_ context.Context, userID, teamID string, since int64, page, perPage int) (*insights.TopInactiveChannelList, error) {
	s.TopInactiveChannelsForUserCalls = append(s.TopInactiveChannelsForUserCalls, InactiveChannelsForUserCall{userID, teamID, since, page, perPage})
	return s.TopInactiveChannelsForUserResult, s.TopInactiveChannelsForUserErr
}

func (s *StoreStub) TopInactiveChannelsForTeamSince(_ context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopInactiveChannelList, error) {
	s.TopInactiveChannelsForTeamCalls = append(s.TopInactiveChannelsForTeamCalls, InactiveChannelsForTeamCall{teamID, userID, since, page, perPage})
	return s.TopInactiveChannelsForTeamResult, s.TopInactiveChannelsForTeamErr
}

func (s *StoreStub) TopDMsForUserSince(_ context.Context, userID string, since int64, page, perPage int) (*insights.TopDMList, error) {
	s.TopDMsForUserCalls = append(s.TopDMsForUserCalls, DMsForUserCall{userID, since, page, perPage})
	return s.TopDMsForUserResult, s.TopDMsForUserErr
}

func (s *StoreStub) OutgoingDMCounts(_ context.Context, userID string, channelIDs []string, since int64) (map[string]int64, error) {
	s.OutgoingDMCountsCalls = append(s.OutgoingDMCountsCalls, OutgoingDMCountsCall{userID, channelIDs, since})
	return s.OutgoingDMCountsResult, s.OutgoingDMCountsErr
}

func (s *StoreStub) PostCountsByDuration(_ context.Context, channelIDs []string, since int64, userID, grouping, location string) ([]*insights.DurationPostCount, error) {
	s.PostCountsByDurationCalls = append(s.PostCountsByDurationCalls, PostCountsByDurationCall{channelIDs, since, userID, grouping, location})
	return s.PostCountsByDurationResult, s.PostCountsByDurationErr
}

func (s *StoreStub) BoardIDsForUserInTeam(_ context.Context, userID, teamID string) ([]string, error) {
	s.BoardIDsForUserInTeamCalls = append(s.BoardIDsForUserInTeamCalls, BoardIDsForUserInTeamCall{userID, teamID})
	return s.BoardIDsForUserInTeamResult, s.BoardIDsForUserInTeamErr
}

func (s *StoreStub) TopBoardsForTeam(_ context.Context, teamID string, boardIDs []string, since int64, page, perPage int) (*insights.TopBoardList, error) {
	s.TopBoardsForTeamCalls = append(s.TopBoardsForTeamCalls, BoardsForTeamCall{teamID, boardIDs, since, page, perPage})
	return s.TopBoardsForTeamResult, s.TopBoardsForTeamErr
}

func (s *StoreStub) TopBoardsForUser(_ context.Context, teamID, userID string, boardIDs []string, since int64, page, perPage int) (*insights.TopBoardList, error) {
	s.TopBoardsForUserCalls = append(s.TopBoardsForUserCalls, BoardsForUserCall{teamID, userID, boardIDs, since, page, perPage})
	return s.TopBoardsForUserResult, s.TopBoardsForUserErr
}

func (s *StoreStub) TopPlaybooksForTeam(_ context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopPlaybookList, error) {
	s.TopPlaybooksForTeamCalls = append(s.TopPlaybooksForTeamCalls, PlaybooksForTeamCall{teamID, userID, since, page, perPage})
	return s.TopPlaybooksForTeamResult, s.TopPlaybooksForTeamErr
}

func (s *StoreStub) TopPlaybooksForUser(_ context.Context, teamID, userID string, since int64, page, perPage int) (*insights.TopPlaybookList, error) {
	s.TopPlaybooksForUserCalls = append(s.TopPlaybooksForUserCalls, PlaybooksForUserCall{teamID, userID, since, page, perPage})
	return s.TopPlaybooksForUserResult, s.TopPlaybooksForUserErr
}

// DirectoryStub is an in-memory implementation of api.Directory for tests.
//
// ShowFullNameSetting defaults to true so tests that don't care about the
// privacy gate get the more permissive behavior.
type DirectoryStub struct {
	Users               map[string]*model.User
	Posts               map[string]*model.Post
	ShowFullNameSetting *bool

	ListUsersByIDsErr error
	GetPostErr        error
}

func (d *DirectoryStub) ListUsersByIDs(userIDs []string) ([]*model.User, error) {
	if d.ListUsersByIDsErr != nil {
		return nil, d.ListUsersByIDsErr
	}
	out := make([]*model.User, 0, len(userIDs))
	for _, id := range userIDs {
		if u, ok := d.Users[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}

func (d *DirectoryStub) GetPost(postID string) (*model.Post, error) {
	if d.GetPostErr != nil {
		return nil, d.GetPostErr
	}
	if p, ok := d.Posts[postID]; ok {
		return p, nil
	}
	return nil, errPostNotFound
}

var errPostNotFound = &model.AppError{Message: "post not found", StatusCode: 404}

func (d *DirectoryStub) ShowFullName() bool {
	if d.ShowFullNameSetting == nil {
		return true
	}
	return *d.ShowFullNameSetting
}
