import type {
    ChannelPostCountByDuration,
    NewTeamMember,
    TimeRange,
    TopBoard,
    TopChannel,
    TopDM,
    TopInactiveChannel,
    TopPlaybook,
    TopReaction,
    TopThread,
} from '../types';

export type Status = 'idle' | 'loading' | 'error';

export interface SliceEntry<T> {
    items: T[];
    hasNext: boolean;
    status: Status;
    error?: string;
    fetchedAt?: number;
}

// Scope buckets are keyed by id: team scope by team id, my scope by team id
// or 'global' when there is no team filter.
type ScopedSlice<T> = Record<string, Record<TimeRange, SliceEntry<T>>>;

export interface TopChannelsState {
    team: ScopedSlice<TopChannel>;
    my: ScopedSlice<TopChannel>;
    postCountByDuration: ChannelPostCountByDuration;
}

export interface NewTeamMembersEntry extends SliceEntry<NewTeamMember> {
    totalCount: number;
}

export interface InsightsState {
    topReactions: {
        team: ScopedSlice<TopReaction>;
        my: ScopedSlice<TopReaction>;
    };
    topChannels: TopChannelsState;
    topThreads: {
        team: ScopedSlice<TopThread>;
        my: ScopedSlice<TopThread>;
    };
    topDms: {
        my: ScopedSlice<TopDM>;
    };
    topInactiveChannels: {
        team: ScopedSlice<TopInactiveChannel>;
        my: ScopedSlice<TopInactiveChannel>;
    };
    newTeamMembers: {
        team: Record<string, Record<TimeRange, NewTeamMembersEntry>>;
    };
    topPlaybooks: {
        team: ScopedSlice<TopPlaybook>;
        my: ScopedSlice<TopPlaybook>;
    };
    topBoards: {
        team: ScopedSlice<TopBoard>;
        my: ScopedSlice<TopBoard>;
    };
}
