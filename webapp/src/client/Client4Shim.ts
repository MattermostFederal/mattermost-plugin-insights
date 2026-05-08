// Compatibility shim that publishes the deprecated `Client4` insights
// methods on a `window.PluginInsightsClient` global. The deprecated
// `mattermost-redux/Client4` shipped these methods (see
// webapp/platform/client/src/client4.ts in commit 26617fcbdc^):
//
//   - getTopReactionsForTeam(teamId, page, perPage, timeRange)
//   - getMyTopReactions(teamId, page, perPage, timeRange)
//   - getTopChannelsForTeam(teamId, page, perPage, timeRange)
//   - getMyTopChannels(teamId, page, perPage, timeRange)
//   - getTopThreadsForTeam(teamId, page, perPage, timeRange)
//   - getMyTopThreads(teamId, page, perPage, timeRange)
//   - getLeastActiveChannelsForTeam(teamId, page, perPage, timeRange)
//   - getMyLeastActiveChannels(teamId, page, perPage, timeRange)
//   - getMyTopDMs(teamId, page, perPage, timeRange)
//   - getNewTeamMembers(teamId, page, perPage, timeRange)
//
// Other plugins / desktop integrations / browser bookmarklets that
// hard-coded those names against the host Client4 will get a 404 today
// because the host removed them. This shim re-exposes the same names
// pointing at the plugin's own server routes (and at the Boards/Playbooks
// plugins for those two), so the deprecated callsite shape keeps working.
//
// Method signatures match the deprecated Client4 verbatim — same
// argument order, same Promise return type. Field names on responses
// match the deprecated TopReactionResponse / TopChannelResponse / etc
// shapes (which are what our server returns anyway).

import {Client} from './Client';

export interface PluginInsightsClient {
    getTopReactionsForTeam(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getMyTopReactions(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getTopChannelsForTeam(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getMyTopChannels(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getTopThreadsForTeam(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getMyTopThreads(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getLeastActiveChannelsForTeam(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getMyLeastActiveChannels(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getMyTopDMs(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getNewTeamMembers(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getTopBoardsForTeam(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getMyTopBoards(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getTopPlaybooksForTeam(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
    getMyTopPlaybooks(teamId: string, page: number, perPage: number, timeRange: string): Promise<unknown>;
}

interface HostWindow extends Window {
    PluginInsightsClient?: PluginInsightsClient;
}

export function installClient4Shim(): void {
    if (typeof window === 'undefined') {
        return;
    }
    const w = window as HostWindow;
    if (w.PluginInsightsClient) {
        return;
    }
    const tr = (timeRange: string) => timeRange as Parameters<typeof Client.getTopReactionsForTeam>[1];
    w.PluginInsightsClient = {
        getTopReactionsForTeam: (teamId, page, perPage, timeRange) =>
            Client.getTopReactionsForTeam(teamId, tr(timeRange), {page, perPage}),
        getMyTopReactions: (teamId, page, perPage, timeRange) =>
            Client.getMyTopReactions(tr(timeRange), {page, perPage, teamId}),
        getTopChannelsForTeam: (teamId, page, perPage, timeRange) =>
            Client.getTopChannelsForTeam(teamId, tr(timeRange), {page, perPage}),
        getMyTopChannels: (teamId, page, perPage, timeRange) =>
            Client.getMyTopChannels(tr(timeRange), {page, perPage, teamId}),
        getTopThreadsForTeam: (teamId, page, perPage, timeRange) =>
            Client.getTopThreadsForTeam(teamId, tr(timeRange), {page, perPage}),
        getMyTopThreads: (teamId, page, perPage, timeRange) =>
            Client.getMyTopThreads(tr(timeRange), {page, perPage, teamId}),
        getLeastActiveChannelsForTeam: (teamId, page, perPage, timeRange) =>
            Client.getTopInactiveChannelsForTeam(teamId, tr(timeRange), {page, perPage}),
        getMyLeastActiveChannels: (teamId, page, perPage, timeRange) =>
            Client.getMyTopInactiveChannels(tr(timeRange), {page, perPage, teamId}),
        getMyTopDMs: (_teamId, page, perPage, timeRange) =>
            Client.getMyTopDMs(tr(timeRange), {page, perPage}),
        getNewTeamMembers: (teamId, page, perPage, timeRange) =>
            Client.getNewTeamMembers(teamId, tr(timeRange), {page, perPage}),
        getTopBoardsForTeam: (teamId, page, perPage, timeRange) =>
            Client.getTopBoardsForTeam(teamId, tr(timeRange), {page, perPage}),
        getMyTopBoards: (teamId, page, perPage, timeRange) =>
            Client.getMyTopBoards(tr(timeRange), {page, perPage, teamId}),
        getTopPlaybooksForTeam: (teamId, page, perPage, timeRange) =>
            Client.getTopPlaybooksForTeam(teamId, tr(timeRange), {page, perPage}),
        getMyTopPlaybooks: (teamId, page, perPage, timeRange) =>
            Client.getMyTopPlaybooks(tr(timeRange), {page, perPage, teamId}),
    };
}
