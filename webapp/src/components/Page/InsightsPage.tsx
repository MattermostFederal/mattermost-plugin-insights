import React, {useCallback, useEffect, useState} from 'react';
import {useSelector} from 'react-redux';

import {ENABLE_PERSONAL_INSIGHTS} from '../../config';
import {useEffectiveScope} from '../../hooks/useLicenseChecks';
import {getCurrentTeamId} from '../../redux/mmSelectors';
import type {Scope, TimeRange} from '../../types';
import {trackInsightsEvent} from '../../utils/telemetry';
import {InsightCard} from '../Card/InsightCard';
import {NewTeamMembersCard} from '../Cards/NewTeamMembersCard';
import {TopBoardsCard} from '../Cards/TopBoardsCard';
import {TopChannelsCard} from '../Cards/TopChannelsCard';
import {TopDMsCard} from '../Cards/TopDMsCard';
import {TopInactiveChannelsCard} from '../Cards/TopInactiveChannelsCard';
import {TopPlaybooksCard} from '../Cards/TopPlaybooksCard';
import {TopReactionsCard} from '../Cards/TopReactionsCard';
import {TopThreadsCard} from '../Cards/TopThreadsCard';
import {ScopeSelect} from '../Controls/ScopeSelect';
import {TimeRangeSelect} from '../Controls/TimeRangeSelect';
import type {InsightsWidgetType} from '../Modal/InsightsModal';
import {InsightsModal} from '../Modal/InsightsModal';

// With ENABLE_PERSONAL_INSIGHTS off there is only one scope, so the URL's
// `scope` param is ignored rather than honored-then-overridden.
const defaultScope: Scope = ENABLE_PERSONAL_INSIGHTS ? 'my' : 'team';

function readQueryState(): {scope: Scope; range: TimeRange} {
    if (typeof window === 'undefined') {
        return {scope: defaultScope, range: '7_day'};
    }
    const params = new URLSearchParams(window.location.search);
    let scope: Scope = 'team';
    if (ENABLE_PERSONAL_INSIGHTS && params.get('scope') !== 'team') {
        scope = 'my';
    }
    const rawRange = params.get('range');
    const range: TimeRange = (rawRange === 'today' || rawRange === '28_day') ? rawRange : '7_day';
    return {scope, range};
}

function writeQueryState(scope: Scope, range: TimeRange) {
    if (typeof window === 'undefined') {
        return;
    }
    const params = new URLSearchParams(window.location.search);
    params.set('scope', scope);
    params.set('range', range);
    const next = `${window.location.pathname}?${params.toString()}`;
    window.history.replaceState(null, '', next);
}

interface CardSpec {
    key: InsightsWidgetType;
    title: string;
    subtitle: string;
    teamOnly?: boolean;
    userOnly?: boolean;
}

const cards: CardSpec[] = [
    {key: 'topChannels', title: 'Top Channels', subtitle: 'The most active channels'},
    {key: 'topReactions', title: 'Top Reactions', subtitle: 'The most used emoji reactions'},
    {key: 'topThreads', title: 'Top Threads', subtitle: 'The threads with the most replies'},
    {key: 'topDms', title: 'Top DMs', subtitle: 'Your most active direct messages', userOnly: true},
    {key: 'topInactiveChannels', title: 'Top Inactive Channels', subtitle: 'The channels with the least activity'},
    {key: 'topBoards', title: 'Top Boards', subtitle: 'Most active boards for the team'},
    {key: 'topPlaybooks', title: 'Top Playbooks', subtitle: 'Playbooks with the most runs'},
    {key: 'newTeamMembers', title: 'New Team Members', subtitle: 'People who recently joined the team', teamOnly: true},
];

export const InsightsPage: React.FC = () => {
    const initial = readQueryState();
    const [requestedScope, setRequestedScope] = useState<Scope>(initial.scope);
    const [range, setRange] = useState<TimeRange>(initial.range);
    const [modal, setModal] = useState<{widgetType: InsightsWidgetType; title: string; subtitle: string} | null>(null);

    // License gate: deprecated `useGetFilterType` forces MY scope on
    // starter-free / non-enterprise. Honor that even when the URL or
    // explicit user click set scope=team.
    //
    // With personal insights disabled that fallback has nowhere to land — the
    // My routes are unregistered — so scope is pinned to team and those
    // servers get an empty page. See ENABLE_PERSONAL_INSIGHTS.
    const licensedScope = useEffectiveScope(requestedScope);
    const scope: Scope = ENABLE_PERSONAL_INSIGHTS ? licensedScope : 'team';

    const teamId = useSelector(getCurrentTeamId);

    useEffect(() => {
        writeQueryState(scope, range);
    }, [scope, range]);

    const handleScopeChange = useCallback((next: Scope) => {
        trackInsightsEvent(next === 'team' ? 'change_scope_to_team_insights' : 'change_scope_to_my_insights');
        setRequestedScope(next);
    }, []);
    const handleRangeChange = useCallback((next: TimeRange) => {
        trackInsightsEvent(`time_frame_selected_${next}`);
        setRange(next);
    }, []);

    const openDetails = useCallback((spec: CardSpec) => {
        trackInsightsEvent(`open_modal_${spec.key.toLowerCase()}`);
        setModal({widgetType: spec.key, title: spec.title, subtitle: spec.subtitle});
    }, []);

    const closeDetails = useCallback(() => setModal(null), []);

    const visible = cards.filter((c) => {
        if (scope === 'team' && c.userOnly) {
            return false;
        }
        if (scope === 'my' && c.teamOnly) {
            return false;
        }
        return true;
    });

    return (
        <main className='insights-page'>
            <header className='insights-page__controls'>
                {ENABLE_PERSONAL_INSIGHTS && (
                    <ScopeSelect
                        value={scope}
                        onChange={handleScopeChange}
                    />
                )}
                <TimeRangeSelect
                    value={range}
                    onChange={handleRangeChange}
                />
            </header>
            <div className='insights-page__grid'>
                {visible.map((c) => {
                    const handleOpen = () => openDetails(c);
                    if (c.key === 'topChannels') {
                        return (
                            <TopChannelsCard
                                key={c.key}
                                scope={scope}
                                timeRange={range}
                                teamId={teamId}
                                onOpenDetails={handleOpen}
                            />
                        );
                    }
                    if (c.key === 'topReactions') {
                        return (
                            <TopReactionsCard
                                key={c.key}
                                scope={scope}
                                timeRange={range}
                                teamId={teamId}
                                onOpenDetails={handleOpen}
                            />
                        );
                    }
                    if (c.key === 'topThreads') {
                        return (
                            <TopThreadsCard
                                key={c.key}
                                scope={scope}
                                timeRange={range}
                                teamId={teamId}
                                onOpenDetails={handleOpen}
                            />
                        );
                    }
                    if (c.key === 'topDms') {
                        return (
                            <TopDMsCard
                                key={c.key}
                                teamId={teamId}
                                timeRange={range}
                                onOpenDetails={handleOpen}
                            />
                        );
                    }
                    if (c.key === 'topInactiveChannels') {
                        return (
                            <TopInactiveChannelsCard
                                key={c.key}
                                scope={scope}
                                timeRange={range}
                                teamId={teamId}
                                onOpenDetails={handleOpen}
                            />
                        );
                    }
                    if (c.key === 'topBoards') {
                        return (
                            <TopBoardsCard
                                key={c.key}
                                scope={scope}
                                timeRange={range}
                                teamId={teamId}
                                onOpenDetails={handleOpen}
                            />
                        );
                    }
                    if (c.key === 'topPlaybooks') {
                        return (
                            <TopPlaybooksCard
                                key={c.key}
                                scope={scope}
                                timeRange={range}
                                teamId={teamId}
                                onOpenDetails={handleOpen}
                            />
                        );
                    }
                    if (c.key === 'newTeamMembers') {
                        return (
                            <NewTeamMembersCard
                                key={c.key}
                                teamId={teamId}
                                timeRange={range}
                                onOpenDetails={handleOpen}
                            />
                        );
                    }
                    return (
                        <InsightCard
                            key={c.key}
                            title={c.title}
                        />
                    );
                })}
            </div>
            {modal ? (
                <InsightsModal
                    show={true}
                    onExited={closeDetails}
                    widgetType={modal.widgetType}
                    title={modal.title}
                    subtitle={modal.subtitle}
                    scope={scope}
                    timeRange={range}
                    teamId={teamId}
                />
            ) : null}
        </main>
    );
};
