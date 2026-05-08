export type Scope = 'team' | 'my';

export type TimeRange = 'today' | '7_day' | '28_day';

export const TIME_RANGES: TimeRange[] = ['today', '7_day', '28_day'];
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
