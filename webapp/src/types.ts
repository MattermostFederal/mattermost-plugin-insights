export type Scope = 'team' | 'my';

// 'today' was retired because "since midnight" is a moving, partial window
// that a once-daily snapshot cannot answer coherently. '1_day' replaces it and
// means *yesterday* — a closed, complete UTC day. Because that window has
// already ended, one snapshot of it stays correct all day, which is what keeps
// the cache to a single daily rebuild.
//
// The cost: today's activity does not appear anywhere until tomorrow.
// See server/insights/timerange.go (WindowUTC).
export type TimeRange = '1_day' | '7_day' | '28_day';

export const TIME_RANGES: TimeRange[] = ['1_day', '7_day', '28_day'];
export const SCOPES: Scope[] = ['my', 'team'];

export interface TopReaction {
    emoji_name: string;
    count: number;
}

export interface TopChannel {
    id: string;
    type: string;
    display_name: string;
    name: string;
    team_id: string;
    message_count: number;
}

// ChannelActivity backs the channel-governance table. Unlike TopChannel it
// carries the metadata columns and is returned for every channel in the team,
// including ones with no activity — a channel with many members and zero
// posts is what the table exists to surface.
export interface ChannelActivity {
    id: string;
    type: string;
    display_name: string;
    name: string;
    purpose: string;
    header: string;
    create_at: number;
    last_post_at: number;
    last_post_in_window: number;
    message_count: number;
    active_posters: number;
    member_count: number;
}

// Summary counts describe the whole team, not the current page — the server
// computes them because the client only ever holds one page.
export interface ChannelGovernanceSummary {
    total_channels: number;
    active_channels: number;
    with_purpose: number;
    with_header: number;

    // How many rows survived the search/filter. The other counts describe the
    // whole team regardless, so searching does not move the denominators.
    matching_channels: number;
}

export interface TopThread {
    channel_id: string;
    channel_display_name: string;
    channel_name: string;
    participants: string[];
    user_information: {
        id: string;
        username: string;
        first_name: string;
        last_name: string;
        nickname: string;
        last_picture_update: number;
    };
    post: {id?: string} & Record<string, unknown>;
}

export interface TopInactiveChannel {
    id: string;
    type: string;
    display_name: string;
    name: string;
    last_activity_at: number;
    participants: string[];
}

export interface TopDM {
    post_count: number;
    outgoing_message_count: number;
    second_participant: {
        id: string;
        username: string;
        first_name: string;
        last_name: string;
        nickname: string;
        position: string;
        last_picture_update: number;
    };
}

export interface NewTeamMember {
    id: string;
    username: string;
    first_name: string;
    last_name: string;
    position: string;
    nickname: string;
    last_picture_update?: number;
    create_at: number;
}

// TopBoard mirrors the deprecated focalboard `BoardInsight` JSON shape
// (mattermost-plugin-boards/server/model/board_insights.go pre-c8e729b6).
// Field names match the deprecated JSON contract verbatim so any downstream
// consumer of the original Insights endpoint receives an identical-looking
// response.
export interface TopBoard {
    boardID: string;
    icon: string;
    title: string;
    activityCount: string;
    activeUsers: string[];
    createdBy: string;
}

// TopPlaybook mirrors the Playbooks plugin's `PlaybookInsight` JSON shape
// (server/app/playbook.go in mattermost-plugin-playbooks). The Playbooks
// plugin server still serves this insight at
// `/plugins/playbooks/api/v0/playbooks/insights/...` (audited 2026-05-08).
export interface TopPlaybook {
    playbook_id: string;
    num_runs: number;
    title: string;
    last_run_at: number;
}

export type ChannelPostCountByDuration = Record<string, Record<string, number>>;

export interface PaginatedResponse<T> {
    has_next: boolean;
    items: T[];
}

export type TopReactionsResponse = PaginatedResponse<TopReaction>;
export type TopChannelsResponse = PaginatedResponse<TopChannel> & {
    channel_post_counts_by_duration: ChannelPostCountByDuration;
};
export type TopThreadsResponse = PaginatedResponse<TopThread>;
export type TopInactiveChannelsResponse = PaginatedResponse<TopInactiveChannel>;
export type TopDMsResponse = PaginatedResponse<TopDM>;
export type NewTeamMembersResponse = PaginatedResponse<NewTeamMember> & {
    total_count: number;
};
export type TopPlaybooksResponse = PaginatedResponse<TopPlaybook>;
export type TopBoardsResponse = PaginatedResponse<TopBoard>;
export type ChannelGovernanceResponse = PaginatedResponse<ChannelActivity> & {
    summary: ChannelGovernanceSummary;

    // When the snapshot was built, unix millis. Surfaced because the numbers
    // are up to a day old by design.
    generated_at: number;
};
