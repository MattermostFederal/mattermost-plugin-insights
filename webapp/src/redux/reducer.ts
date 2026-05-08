import type {Reducer} from 'redux';

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
import type {InsightsState, NewTeamMembersEntry, SliceEntry, TopChannelsState} from './types';

import type {ChannelPostCountByDuration, NewTeamMember, Scope, TimeRange, TopBoard, TopChannel, TopDM, TopInactiveChannel, TopPlaybook, TopReaction, TopThread} from '../types';

const emptyState: InsightsState = {
    topReactions: {team: {}, my: {}},
    topChannels: {team: {}, my: {}, postCountByDuration: {}},
    topThreads: {team: {}, my: {}},
    topDms: {my: {}},
    topInactiveChannels: {team: {}, my: {}},
    newTeamMembers: {team: {}},
    topPlaybooks: {team: {}, my: {}},
    topBoards: {team: {}, my: {}},
};

function emptyEntry<T>(): SliceEntry<T> {
    return {items: [], hasNext: false, status: 'idle'};
}

function setEntry<T>(
    bucket: Record<string, Record<TimeRange, SliceEntry<T>>>,
    scopeKey: string,
    timeRange: TimeRange,
    next: SliceEntry<T>,
): Record<string, Record<TimeRange, SliceEntry<T>>> {
    const existing = bucket[scopeKey] ?? ({} as Record<TimeRange, SliceEntry<T>>);
    return {...bucket, [scopeKey]: {...existing, [timeRange]: next}};
}

function applyLoading<T>(
    state: {team: Record<string, Record<TimeRange, SliceEntry<T>>>; my: Record<string, Record<TimeRange, SliceEntry<T>>>},
    p: {scope: Scope; scopeKey: string; timeRange: TimeRange},
): typeof state {
    const prev = state[p.scope][p.scopeKey]?.[p.timeRange] ?? emptyEntry<T>();
    return {...state, [p.scope]: setEntry(state[p.scope], p.scopeKey, p.timeRange, {...prev, status: 'loading', error: undefined})};
}

function applyReceive<T>(
    state: {team: Record<string, Record<TimeRange, SliceEntry<T>>>; my: Record<string, Record<TimeRange, SliceEntry<T>>>},
    p: {scope: Scope; scopeKey: string; timeRange: TimeRange; items: T[]; hasNext: boolean},
): typeof state {
    return {...state, [p.scope]: setEntry(state[p.scope], p.scopeKey, p.timeRange, {items: p.items, hasNext: p.hasNext, status: 'idle', fetchedAt: Date.now()})};
}

function applyError<T>(
    state: {team: Record<string, Record<TimeRange, SliceEntry<T>>>; my: Record<string, Record<TimeRange, SliceEntry<T>>>},
    p: {scope: Scope; scopeKey: string; timeRange: TimeRange; error: string},
): typeof state {
    const prev = state[p.scope][p.scopeKey]?.[p.timeRange] ?? emptyEntry<T>();
    return {...state, [p.scope]: setEntry(state[p.scope], p.scopeKey, p.timeRange, {...prev, status: 'error', error: p.error})};
}

function setNewMembersEntry(
    bucket: Record<string, Record<TimeRange, NewTeamMembersEntry>>,
    scopeKey: string,
    timeRange: TimeRange,
    next: NewTeamMembersEntry,
): Record<string, Record<TimeRange, NewTeamMembersEntry>> {
    const existing = bucket[scopeKey] ?? ({} as Record<TimeRange, NewTeamMembersEntry>);
    return {...bucket, [scopeKey]: {...existing, [timeRange]: next}};
}

function newMembersEmpty(): NewTeamMembersEntry {
    return {items: [], hasNext: false, status: 'idle', totalCount: 0};
}

const reducer: Reducer<InsightsState> = (state = emptyState, action: {type: string; payload?: unknown} = {type: ''}) => {
    switch (action.type) {
    case LOADING_TOP_REACTIONS:
        return {...state, topReactions: applyLoading<TopReaction>(state.topReactions, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange})};
    case RECEIVE_TOP_REACTIONS:
        return {...state, topReactions: applyReceive<TopReaction>(state.topReactions, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; items: TopReaction[]; hasNext: boolean})};
    case ERROR_TOP_REACTIONS:
        return {...state, topReactions: applyError<TopReaction>(state.topReactions, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; error: string})};
    case LOADING_TOP_THREADS:
        return {...state, topThreads: applyLoading<TopThread>(state.topThreads, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange})};
    case RECEIVE_TOP_THREADS:
        return {...state, topThreads: applyReceive<TopThread>(state.topThreads, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; items: TopThread[]; hasNext: boolean})};
    case ERROR_TOP_THREADS:
        return {...state, topThreads: applyError<TopThread>(state.topThreads, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; error: string})};

    case LOADING_TOP_CHANNELS: {
        const p = action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange};
        const next = applyLoading<TopChannel>({team: state.topChannels.team, my: state.topChannels.my}, p);
        return {...state, topChannels: {...state.topChannels, ...next} as TopChannelsState};
    }
    case RECEIVE_TOP_CHANNELS: {
        const p = action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; items: TopChannel[]; hasNext: boolean; postCountByDuration?: ChannelPostCountByDuration};
        const next = applyReceive<TopChannel>({team: state.topChannels.team, my: state.topChannels.my}, p);
        return {...state, topChannels: {...state.topChannels, ...next, postCountByDuration: p.postCountByDuration ?? state.topChannels.postCountByDuration}};
    }
    case ERROR_TOP_CHANNELS: {
        const p = action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; error: string};
        const next = applyError<TopChannel>({team: state.topChannels.team, my: state.topChannels.my}, p);
        return {...state, topChannels: {...state.topChannels, ...next} as TopChannelsState};
    }

    case LOADING_TOP_INACTIVE_CHANNELS:
        return {...state, topInactiveChannels: applyLoading<TopInactiveChannel>(state.topInactiveChannels, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange})};
    case RECEIVE_TOP_INACTIVE_CHANNELS:
        return {...state, topInactiveChannels: applyReceive<TopInactiveChannel>(state.topInactiveChannels, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; items: TopInactiveChannel[]; hasNext: boolean})};
    case ERROR_TOP_INACTIVE_CHANNELS:
        return {...state, topInactiveChannels: applyError<TopInactiveChannel>(state.topInactiveChannels, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; error: string})};

    case LOADING_TOP_DMS: {
        const p = action.payload as {scopeKey: string; timeRange: TimeRange};
        const prev = state.topDms.my[p.scopeKey]?.[p.timeRange] ?? emptyEntry<TopDM>();
        return {...state, topDms: {my: setEntry(state.topDms.my, p.scopeKey, p.timeRange, {...prev, status: 'loading', error: undefined})}};
    }
    case RECEIVE_TOP_DMS: {
        const p = action.payload as {scopeKey: string; timeRange: TimeRange; items: TopDM[]; hasNext: boolean};
        return {...state, topDms: {my: setEntry(state.topDms.my, p.scopeKey, p.timeRange, {items: p.items, hasNext: p.hasNext, status: 'idle', fetchedAt: Date.now()})}};
    }
    case ERROR_TOP_DMS: {
        const p = action.payload as {scopeKey: string; timeRange: TimeRange; error: string};
        const prev = state.topDms.my[p.scopeKey]?.[p.timeRange] ?? emptyEntry<TopDM>();
        return {...state, topDms: {my: setEntry(state.topDms.my, p.scopeKey, p.timeRange, {...prev, status: 'error', error: p.error})}};
    }

    case LOADING_TOP_PLAYBOOKS:
        return {...state, topPlaybooks: applyLoading<TopPlaybook>(state.topPlaybooks, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange})};
    case RECEIVE_TOP_PLAYBOOKS:
        return {...state, topPlaybooks: applyReceive<TopPlaybook>(state.topPlaybooks, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; items: TopPlaybook[]; hasNext: boolean})};
    case ERROR_TOP_PLAYBOOKS:
        return {...state, topPlaybooks: applyError<TopPlaybook>(state.topPlaybooks, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; error: string})};

    case LOADING_TOP_BOARDS:
        return {...state, topBoards: applyLoading<TopBoard>(state.topBoards, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange})};
    case RECEIVE_TOP_BOARDS:
        return {...state, topBoards: applyReceive<TopBoard>(state.topBoards, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; items: TopBoard[]; hasNext: boolean})};
    case ERROR_TOP_BOARDS:
        return {...state, topBoards: applyError<TopBoard>(state.topBoards, action.payload as {scope: Scope; scopeKey: string; timeRange: TimeRange; error: string})};

    case LOADING_NEW_TEAM_MEMBERS: {
        const p = action.payload as {scopeKey: string; timeRange: TimeRange};
        const prev = state.newTeamMembers.team[p.scopeKey]?.[p.timeRange] ?? newMembersEmpty();
        return {...state, newTeamMembers: {team: setNewMembersEntry(state.newTeamMembers.team, p.scopeKey, p.timeRange, {...prev, status: 'loading', error: undefined})}};
    }
    case RECEIVE_NEW_TEAM_MEMBERS: {
        const p = action.payload as {scopeKey: string; timeRange: TimeRange; items: NewTeamMember[]; hasNext: boolean; totalCount: number};
        return {...state, newTeamMembers: {team: setNewMembersEntry(state.newTeamMembers.team, p.scopeKey, p.timeRange, {items: p.items, hasNext: p.hasNext, status: 'idle', fetchedAt: Date.now(), totalCount: p.totalCount})}};
    }
    case ERROR_NEW_TEAM_MEMBERS: {
        const p = action.payload as {scopeKey: string; timeRange: TimeRange; error: string};
        const prev = state.newTeamMembers.team[p.scopeKey]?.[p.timeRange] ?? newMembersEmpty();
        return {...state, newTeamMembers: {team: setNewMembersEntry(state.newTeamMembers.team, p.scopeKey, p.timeRange, {...prev, status: 'error', error: p.error})}};
    }

    default:
        return state;
    }
};

export default reducer;
