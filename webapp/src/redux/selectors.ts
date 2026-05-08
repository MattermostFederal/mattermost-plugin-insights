import manifest from 'manifest';

import type {InsightsState, NewTeamMembersEntry, SliceEntry} from './types';

import type {ChannelPostCountByDuration, Scope, TimeRange, TopBoard, TopChannel, TopDM, TopInactiveChannel, TopPlaybook, TopReaction, TopThread} from '../types';

interface RootState {
    [key: string]: unknown;
}

const stateKey = `plugins-${manifest.id}`;

export function selectInsightsState(state: RootState): InsightsState | undefined {
    return state[stateKey] as InsightsState | undefined;
}

const idleEntry = <T>(): SliceEntry<T> => ({items: [], hasNext: false, status: 'idle'});

export function selectTopReactions(
    state: RootState,
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
): SliceEntry<TopReaction> {
    return selectInsightsState(state)?.topReactions[scope][scopeKey]?.[timeRange] ?? idleEntry<TopReaction>();
}

export function selectTopThreads(
    state: RootState,
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
): SliceEntry<TopThread> {
    return selectInsightsState(state)?.topThreads[scope][scopeKey]?.[timeRange] ?? idleEntry<TopThread>();
}

export function selectTopChannels(
    state: RootState,
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
): SliceEntry<TopChannel> {
    return selectInsightsState(state)?.topChannels[scope][scopeKey]?.[timeRange] ?? idleEntry<TopChannel>();
}

export function selectChannelPostCountByDuration(state: RootState): ChannelPostCountByDuration {
    return selectInsightsState(state)?.topChannels.postCountByDuration ?? {};
}

export function selectTopInactiveChannels(
    state: RootState,
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
): SliceEntry<TopInactiveChannel> {
    return selectInsightsState(state)?.topInactiveChannels[scope][scopeKey]?.[timeRange] ?? idleEntry<TopInactiveChannel>();
}

export function selectTopDMs(
    state: RootState,
    scopeKey: string,
    timeRange: TimeRange,
): SliceEntry<TopDM> {
    return selectInsightsState(state)?.topDms.my[scopeKey]?.[timeRange] ?? idleEntry<TopDM>();
}

export function selectNewTeamMembers(
    state: RootState,
    teamId: string,
    timeRange: TimeRange,
): NewTeamMembersEntry {
    return selectInsightsState(state)?.newTeamMembers.team[teamId]?.[timeRange] ?? {items: [], hasNext: false, status: 'idle', totalCount: 0};
}

export function selectTopPlaybooks(
    state: RootState,
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
): SliceEntry<TopPlaybook> {
    return selectInsightsState(state)?.topPlaybooks[scope][scopeKey]?.[timeRange] ?? idleEntry<TopPlaybook>();
}

export function selectTopBoards(
    state: RootState,
    scope: Scope,
    scopeKey: string,
    timeRange: TimeRange,
): SliceEntry<TopBoard> {
    return selectInsightsState(state)?.topBoards[scope][scopeKey]?.[timeRange] ?? idleEntry<TopBoard>();
}
