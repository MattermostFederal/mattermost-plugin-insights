import {
    ERROR_NEW_TEAM_MEMBERS,
    ERROR_TOP_BOARDS,
    ERROR_TOP_CHANNELS,
    ERROR_TOP_DMS,
    ERROR_TOP_INACTIVE_CHANNELS,
    ERROR_TOP_PLAYBOOKS,
    ERROR_TOP_REACTIONS,
    ERROR_TOP_THREADS,
    LOADING_NEW_TEAM_MEMBERS,
    LOADING_TOP_BOARDS,
    LOADING_TOP_CHANNELS,
    LOADING_TOP_DMS,
    LOADING_TOP_INACTIVE_CHANNELS,
    LOADING_TOP_PLAYBOOKS,
    LOADING_TOP_REACTIONS,
    LOADING_TOP_THREADS,
    RECEIVE_NEW_TEAM_MEMBERS,
    RECEIVE_TOP_BOARDS,
    RECEIVE_TOP_CHANNELS,
    RECEIVE_TOP_DMS,
    RECEIVE_TOP_INACTIVE_CHANNELS,
    RECEIVE_TOP_PLAYBOOKS,
    RECEIVE_TOP_REACTIONS,
    RECEIVE_TOP_THREADS,
} from './actionTypes';

import {Client} from '../client/Client';
import type {ChannelPostCountByDuration, NewTeamMember, Scope, TimeRange, TopBoard, TopChannel, TopDM, TopInactiveChannel, TopPlaybook, TopReaction, TopThread} from '../types';

interface ScopeKey {
    scope: Scope;
    scopeKey: string;
    timeRange: TimeRange;
}

export interface ReceiveTopReactionsAction {
    type: typeof RECEIVE_TOP_REACTIONS;
    payload: ScopeKey & {items: TopReaction[]; hasNext: boolean};
}

export interface LoadingTopReactionsAction {
    type: typeof LOADING_TOP_REACTIONS;
    payload: ScopeKey;
}

export interface ErrorTopReactionsAction {
    type: typeof ERROR_TOP_REACTIONS;
    payload: ScopeKey & {error: string};
}

export interface ReceiveTopThreadsAction {
    type: typeof RECEIVE_TOP_THREADS;
    payload: ScopeKey & {items: TopThread[]; hasNext: boolean};
}

export interface LoadingTopThreadsAction {
    type: typeof LOADING_TOP_THREADS;
    payload: ScopeKey;
}

export interface ErrorTopThreadsAction {
    type: typeof ERROR_TOP_THREADS;
    payload: ScopeKey & {error: string};
}

interface TeamScopeKey {
    scopeKey: string;
    timeRange: TimeRange;
}

export interface ReceiveNewTeamMembersAction {
    type: typeof RECEIVE_NEW_TEAM_MEMBERS;
    payload: TeamScopeKey & {items: NewTeamMember[]; hasNext: boolean; totalCount: number};
}

export interface LoadingNewTeamMembersAction {
    type: typeof LOADING_NEW_TEAM_MEMBERS;
    payload: TeamScopeKey;
}

export interface ErrorNewTeamMembersAction {
    type: typeof ERROR_NEW_TEAM_MEMBERS;
    payload: TeamScopeKey & {error: string};
}

export interface ReceiveTopChannelsAction {
    type: typeof RECEIVE_TOP_CHANNELS;
    payload: ScopeKey & {items: TopChannel[]; hasNext: boolean; postCountByDuration?: ChannelPostCountByDuration};
}

export interface LoadingTopChannelsAction {
    type: typeof LOADING_TOP_CHANNELS;
    payload: ScopeKey;
}

export interface ErrorTopChannelsAction {
    type: typeof ERROR_TOP_CHANNELS;
    payload: ScopeKey & {error: string};
}

export interface ReceiveTopInactiveChannelsAction {
    type: typeof RECEIVE_TOP_INACTIVE_CHANNELS;
    payload: ScopeKey & {items: TopInactiveChannel[]; hasNext: boolean};
}

export interface LoadingTopInactiveChannelsAction {
    type: typeof LOADING_TOP_INACTIVE_CHANNELS;
    payload: ScopeKey;
}

export interface ErrorTopInactiveChannelsAction {
    type: typeof ERROR_TOP_INACTIVE_CHANNELS;
    payload: ScopeKey & {error: string};
}

interface UserOnlyKey {
    scopeKey: string;
    timeRange: TimeRange;
}

export interface ReceiveTopDMsAction {
    type: typeof RECEIVE_TOP_DMS;
    payload: UserOnlyKey & {items: TopDM[]; hasNext: boolean};
}

export interface LoadingTopDMsAction {
    type: typeof LOADING_TOP_DMS;
    payload: UserOnlyKey;
}

export interface ErrorTopDMsAction {
    type: typeof ERROR_TOP_DMS;
    payload: UserOnlyKey & {error: string};
}

export interface ReceiveTopPlaybooksAction {
    type: typeof RECEIVE_TOP_PLAYBOOKS;
    payload: ScopeKey & {items: TopPlaybook[]; hasNext: boolean};
}

export interface LoadingTopPlaybooksAction {
    type: typeof LOADING_TOP_PLAYBOOKS;
    payload: ScopeKey;
}

export interface ErrorTopPlaybooksAction {
    type: typeof ERROR_TOP_PLAYBOOKS;
    payload: ScopeKey & {error: string};
}

export interface ReceiveTopBoardsAction {
    type: typeof RECEIVE_TOP_BOARDS;
    payload: ScopeKey & {items: TopBoard[]; hasNext: boolean};
}

export interface LoadingTopBoardsAction {
    type: typeof LOADING_TOP_BOARDS;
    payload: ScopeKey;
}

export interface ErrorTopBoardsAction {
    type: typeof ERROR_TOP_BOARDS;
    payload: ScopeKey & {error: string};
}

export type InsightsAction =
    | ReceiveTopReactionsAction
    | LoadingTopReactionsAction
    | ErrorTopReactionsAction
    | ReceiveTopThreadsAction
    | LoadingTopThreadsAction
    | ErrorTopThreadsAction
    | ReceiveNewTeamMembersAction
    | LoadingNewTeamMembersAction
    | ErrorNewTeamMembersAction
    | ReceiveTopChannelsAction
    | LoadingTopChannelsAction
    | ErrorTopChannelsAction
    | ReceiveTopInactiveChannelsAction
    | LoadingTopInactiveChannelsAction
    | ErrorTopInactiveChannelsAction
    | ReceiveTopDMsAction
    | LoadingTopDMsAction
    | ErrorTopDMsAction
    | ReceiveTopPlaybooksAction
    | LoadingTopPlaybooksAction
    | ErrorTopPlaybooksAction
    | ReceiveTopBoardsAction
    | LoadingTopBoardsAction
    | ErrorTopBoardsAction;

type Dispatch = (action: InsightsAction | ((dispatch: Dispatch) => Promise<void>)) => void;

export const requestTopReactions = (
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
    teamId?: string,
) => async (dispatch: Dispatch) => {
    dispatch({type: LOADING_TOP_REACTIONS, payload: {scope, scopeKey, timeRange}});
    try {
        const result = scope === 'team' ? await Client.getTopReactionsForTeam(scopeKey, timeRange) : await Client.getMyTopReactions(timeRange, {teamId});
        dispatch({type: RECEIVE_TOP_REACTIONS, payload: {scope, scopeKey, timeRange, items: result.items, hasNext: result.has_next}});
    } catch (e) {
        dispatch({type: ERROR_TOP_REACTIONS, payload: {scope, scopeKey, timeRange, error: e instanceof Error ? e.message : String(e)}});
    }
};

export const requestTopThreads = (
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
    teamId?: string,
) => async (dispatch: Dispatch) => {
    dispatch({type: LOADING_TOP_THREADS, payload: {scope, scopeKey, timeRange}});
    try {
        const result = scope === 'team' ? await Client.getTopThreadsForTeam(scopeKey, timeRange) : await Client.getMyTopThreads(timeRange, {teamId});
        dispatch({type: RECEIVE_TOP_THREADS, payload: {scope, scopeKey, timeRange, items: result.items, hasNext: result.has_next}});
    } catch (e) {
        dispatch({type: ERROR_TOP_THREADS, payload: {scope, scopeKey, timeRange, error: e instanceof Error ? e.message : String(e)}});
    }
};

export const requestTopChannels = (
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
    teamId?: string,
) => async (dispatch: Dispatch) => {
    dispatch({type: LOADING_TOP_CHANNELS, payload: {scope, scopeKey, timeRange}});
    try {
        const result = scope === 'team' ? await Client.getTopChannelsForTeam(scopeKey, timeRange) : await Client.getMyTopChannels(timeRange, {teamId});
        dispatch({type: RECEIVE_TOP_CHANNELS, payload: {scope, scopeKey, timeRange, items: result.items, hasNext: result.has_next, postCountByDuration: result.channel_post_counts_by_duration}});
    } catch (e) {
        dispatch({type: ERROR_TOP_CHANNELS, payload: {scope, scopeKey, timeRange, error: e instanceof Error ? e.message : String(e)}});
    }
};

export const requestTopInactiveChannels = (
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
    teamId?: string,
) => async (dispatch: Dispatch) => {
    dispatch({type: LOADING_TOP_INACTIVE_CHANNELS, payload: {scope, scopeKey, timeRange}});
    try {
        const result = scope === 'team' ? await Client.getTopInactiveChannelsForTeam(scopeKey, timeRange) : await Client.getMyTopInactiveChannels(timeRange, {teamId});
        dispatch({type: RECEIVE_TOP_INACTIVE_CHANNELS, payload: {scope, scopeKey, timeRange, items: result.items, hasNext: result.has_next}});
    } catch (e) {
        dispatch({type: ERROR_TOP_INACTIVE_CHANNELS, payload: {scope, scopeKey, timeRange, error: e instanceof Error ? e.message : String(e)}});
    }
};

export const requestTopDMs = (
    scopeKey: string,
    timeRange: TimeRange,
) => async (dispatch: Dispatch) => {
    dispatch({type: LOADING_TOP_DMS, payload: {scopeKey, timeRange}});
    try {
        const result = await Client.getMyTopDMs(timeRange);
        dispatch({type: RECEIVE_TOP_DMS, payload: {scopeKey, timeRange, items: result.items, hasNext: result.has_next}});
    } catch (e) {
        dispatch({type: ERROR_TOP_DMS, payload: {scopeKey, timeRange, error: e instanceof Error ? e.message : String(e)}});
    }
};

export const requestNewTeamMembers = (
    teamId: string,
    timeRange: TimeRange,
) => async (dispatch: Dispatch) => {
    dispatch({type: LOADING_NEW_TEAM_MEMBERS, payload: {scopeKey: teamId, timeRange}});
    try {
        const result = await Client.getNewTeamMembers(teamId, timeRange);
        dispatch({type: RECEIVE_NEW_TEAM_MEMBERS, payload: {scopeKey: teamId, timeRange, items: result.items, hasNext: result.has_next, totalCount: result.total_count}});
    } catch (e) {
        dispatch({type: ERROR_NEW_TEAM_MEMBERS, payload: {scopeKey: teamId, timeRange, error: e instanceof Error ? e.message : String(e)}});
    }
};

export const requestTopPlaybooks = (
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
    teamId: string,
) => async (dispatch: Dispatch) => {
    dispatch({type: LOADING_TOP_PLAYBOOKS, payload: {scope, scopeKey, timeRange}});
    try {
        const result = scope === 'team' ?
            await Client.getTopPlaybooksForTeam(scopeKey, timeRange) :
            await Client.getMyTopPlaybooks(timeRange, {teamId});
        dispatch({type: RECEIVE_TOP_PLAYBOOKS, payload: {scope, scopeKey, timeRange, items: result.items, hasNext: result.has_next}});
    } catch (e) {
        dispatch({type: ERROR_TOP_PLAYBOOKS, payload: {scope, scopeKey, timeRange, error: e instanceof Error ? e.message : String(e)}});
    }
};

export const requestTopBoards = (
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
    teamId: string,
) => async (dispatch: Dispatch) => {
    dispatch({type: LOADING_TOP_BOARDS, payload: {scope, scopeKey, timeRange}});
    try {
        const result = scope === 'team' ?
            await Client.getTopBoardsForTeam(scopeKey, timeRange) :
            await Client.getMyTopBoards(timeRange, {teamId});
        dispatch({type: RECEIVE_TOP_BOARDS, payload: {scope, scopeKey, timeRange, items: result.items, hasNext: result.has_next}});
    } catch (e) {
        dispatch({type: ERROR_TOP_BOARDS, payload: {scope, scopeKey, timeRange, error: e instanceof Error ? e.message : String(e)}});
    }
};
