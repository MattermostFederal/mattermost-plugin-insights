import React, {useCallback, useEffect, useState} from 'react';
import {FormattedMessage} from 'react-intl';
import {useSelector} from 'react-redux';

import {ENABLE_PERSONAL_INSIGHTS} from '../../config';
import {useEffectiveScope} from '../../hooks/useLicenseChecks';
import {getEffectiveTeamId} from '../../redux/mmSelectors';
import type {Scope, TimeRange} from '../../types';
import {trackInsightsEvent} from '../../utils/telemetry';
import {NewTeamMembersTile} from '../Cards/NewTeamMembersTile';
import {ScopeSelect} from '../Controls/ScopeSelect';
import {TimeRangeSelect} from '../Controls/TimeRangeSelect';
import type {InsightsWidgetType} from '../Modal/InsightsModal';
import {InsightsModal} from '../Modal/InsightsModal';
import {ChannelGovernanceTable} from '../Modal/Tables/ChannelGovernanceTable';

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
    let range: TimeRange = '7_day';
    if (rawRange === '28_day' || rawRange === '1_day') {
        range = rawRange;
    }
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

// 1.0 ships a single surface: the channel governance table, with New Team
// Members folded into its summary row as a tile (see NewTeamMembersTile — as
// a full card it out-sized the table it sat under).
//
// The scorecard grid is gone with the cards that filled it:
//
//   - Top Channels and Top Inactive Channels are the governance table's rows
//     sorted two ways. Now that its columns are sortable they add nothing, and
//     dropping Top Channels also retired the sparkline query
//     (INSIGHTS_REFERENCE.md §3.3).
//   - Top Reactions and Top Threads both scope private channels per requester
//     (§2.1, §2.3), so putting them on the shared daily snapshot needs
//     channel-grain entries and a read-time sum — the two most expensive
//     pieces of work left, for cards no requirement asks for. Off behind
//     EnableReactionsAndThreads until there is time to do it properly.
//   - Top Boards and Top Playbooks read tables owned by other plugins.
//   - Top DMs went with personal insights.
//
// New Team Members stays because it is the one card that snapshots cheaply:
// no channel dimension at all, so a single entry per team is correct for
// everyone (§2.6).
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

    const teamId = useSelector(getEffectiveTeamId);

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

    const closeDetails = useCallback(() => setModal(null), []);

    // The New Team Members tile opens the same paginated modal the card's
    // chevron used to, so the full list is still one click away.
    const openNewMembers = useCallback(() => {
        trackInsightsEvent('open_modal_newteammembers');
        setModal({
            widgetType: 'newTeamMembers',
            title: 'New Team Members',
            subtitle: 'People who recently joined the team',
        });
    }, []);

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
            {/*
              * Channel governance sits on the page rather than behind a card,
              * because its value is the full list — the long tail of quiet
              * channels — not a top-N summary that a card could show.
              * Team-scoped only; there is no per-user variant.
              */}
            {scope === 'team' && teamId ? (
                <section className='insights-page__governance'>
                    <h2 className='insights-page__section-title'>
                        <FormattedMessage
                            id='insights.governance.title'
                            defaultMessage='Team activity'
                        />
                    </h2>
                    <ChannelGovernanceTable
                        scope={scope}
                        timeRange={range}
                        teamId={teamId}
                        trailingTile={
                            <NewTeamMembersTile
                                teamId={teamId}
                                timeRange={range}
                                onOpenDetails={openNewMembers}
                            />
                        }
                    />
                </section>
            ) : null}
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
