import type {
    ChannelGovernanceResponse,
    NewTeamMembersResponse,
    TimeRange,
    TopBoardsResponse,
    TopChannelsResponse,
    TopDMsResponse,
    TopInactiveChannelsResponse,
    TopPlaybooksResponse,
    TopReactionsResponse,
    TopThreadsResponse,
} from '../types';

const baseURL = '/plugins/insights/api/v1';

interface PageOpts {
    page?: number;
    perPage?: number;
}

function buildQuery(params: Record<string, string | number | undefined>): string {
    const search = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
        if (value !== undefined && value !== '') {
            search.set(key, String(value));
        }
    }
    const qs = search.toString();
    return qs ? `?${qs}` : '';
}

async function get<T>(path: string): Promise<T> {
    const response = await fetch(baseURL + path, {
        credentials: 'include',
        headers: {
            'X-Requested-With': 'XMLHttpRequest',
        },
    });
    if (!response.ok) {
        throw new Error(`insights request failed: ${response.status} ${response.statusText}`);
    }
    return response.json() as Promise<T>;
}

export const Client = {
    getTopReactionsForTeam(teamId: string, timeRange: TimeRange, opts: PageOpts = {}) {
        return get<TopReactionsResponse>(`/teams/${teamId}/top/reactions${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage})}`);
    },
    getMyTopReactions(timeRange: TimeRange, opts: PageOpts & {teamId?: string} = {}) {
        return get<TopReactionsResponse>(`/users/me/top/reactions${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage, team_id: opts.teamId})}`);
    },
    getTopChannelsForTeam(teamId: string, timeRange: TimeRange, opts: PageOpts = {}) {
        return get<TopChannelsResponse>(`/teams/${teamId}/top/channels${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage})}`);
    },

    // Channel governance: every channel in the team with its activity and
    // metadata, plus team-wide labelling coverage. Team scope only.
    getChannelGovernance(teamId: string, timeRange: TimeRange, opts: PageOpts & {sort?: string; direction?: string} = {}) {
        return get<ChannelGovernanceResponse>(`/teams/${teamId}/channel_activity${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage, sort: opts.sort, direction: opts.direction})}`);
    },
    getMyTopChannels(timeRange: TimeRange, opts: PageOpts & {teamId?: string} = {}) {
        return get<TopChannelsResponse>(`/users/me/top/channels${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage, team_id: opts.teamId})}`);
    },
    getTopThreadsForTeam(teamId: string, timeRange: TimeRange, opts: PageOpts = {}) {
        return get<TopThreadsResponse>(`/teams/${teamId}/top/threads${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage})}`);
    },
    getMyTopThreads(timeRange: TimeRange, opts: PageOpts & {teamId?: string} = {}) {
        return get<TopThreadsResponse>(`/users/me/top/threads${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage, team_id: opts.teamId})}`);
    },
    getTopInactiveChannelsForTeam(teamId: string, timeRange: TimeRange, opts: PageOpts = {}) {
        return get<TopInactiveChannelsResponse>(`/teams/${teamId}/top/inactive_channels${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage})}`);
    },
    getMyTopInactiveChannels(timeRange: TimeRange, opts: PageOpts & {teamId?: string} = {}) {
        return get<TopInactiveChannelsResponse>(`/users/me/top/inactive_channels${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage, team_id: opts.teamId})}`);
    },
    getMyTopDMs(timeRange: TimeRange, opts: PageOpts = {}) {
        return get<TopDMsResponse>(`/users/me/top/dms${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage})}`);
    },
    getNewTeamMembers(teamId: string, timeRange: TimeRange, opts: PageOpts = {}) {
        return get<NewTeamMembersResponse>(`/teams/${teamId}/top/team_members${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage})}`);
    },

    // Top Playbooks endpoints are served by THIS plugin (the Playbooks
    // plugin's own `licenseAndGuestCheck` rejects Enterprise Advanced
    // licenses with a 500 — see server/store/playbook.go for why we
    // query the IR_Playbook / IR_Incident / IR_PlaybookMember tables
    // ourselves rather than calling the Playbooks plugin's HTTP routes).
    getMyTopPlaybooks(timeRange: TimeRange, opts: PageOpts & {teamId: string}) {
        return get<TopPlaybooksResponse>(`/users/me/top/playbooks${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage, team_id: opts.teamId})}`);
    },
    getTopPlaybooksForTeam(teamId: string, timeRange: TimeRange, opts: PageOpts = {}) {
        return get<TopPlaybooksResponse>(`/teams/${teamId}/top/playbooks${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage})}`);
    },

    // Top Boards endpoints are served by THIS plugin (the Boards plugin
    // removed its insights endpoints in commit c8e729b6, June 2024). The
    // plugin's server queries the focalboard tables on the same Mattermost
    // database directly.
    getTopBoardsForTeam(teamId: string, timeRange: TimeRange, opts: PageOpts = {}) {
        return get<TopBoardsResponse>(`/teams/${teamId}/top/boards${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage})}`);
    },
    getMyTopBoards(timeRange: TimeRange, opts: PageOpts & {teamId: string}) {
        return get<TopBoardsResponse>(`/users/me/top/boards${buildQuery({time_range: timeRange, page: opts.page, per_page: opts.perPage, team_id: opts.teamId})}`);
    },
};
