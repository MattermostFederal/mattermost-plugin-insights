// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_playbooks/top_playbooks.tsx
// (commit 26617fcbdc).
//
// Adaptations:
//   - Skeleton blocks moved into a shared `TopPlaybooksSkeleton` (matches
//     the deprecated `useMemo` body exactly).
//   - Row click navigation goes through the plugin's `navigateTo` helper
//     instead of `<Link>` so the host's history is reused without
//     pulling in `react-router-dom`.
//   - `Timestamp` (host-component) → `Intl.RelativeTimeFormat` for the
//     "Last run: …" line.

import React from 'react';
import {FormattedMessage} from 'react-intl';

import type {Status} from '../../redux/types';
import type {TopPlaybook} from '../../types';
import {navigateTo} from '../../utils/navigation';
import {trackInsightsEvent} from '../../utils/telemetry';
import {WidgetEmptyState} from '../EmptyState/WidgetEmptyState';
import {TopPlaybooksSkeleton} from '../Skeleton/CardSkeletons';

interface Props {
    items: TopPlaybook[];
    hasNext: boolean;
    status: Status;
    error?: string;
    teamName?: string;
}

const RELATIVE_UNITS: Array<{limitMs: number; divisor: number; unit: Intl.RelativeTimeFormatUnit}> = [
    {limitMs: 60_000, divisor: 1_000, unit: 'second'},
    {limitMs: 3_600_000, divisor: 60_000, unit: 'minute'},
    {limitMs: 86_400_000, divisor: 3_600_000, unit: 'hour'},
    {limitMs: 30 * 86_400_000, divisor: 86_400_000, unit: 'day'},
    {limitMs: 365 * 86_400_000, divisor: 30 * 86_400_000, unit: 'month'},
];

function relativeTime(unixMillis: number): string {
    if (!unixMillis) {
        return '';
    }
    const fmt = new Intl.RelativeTimeFormat(undefined, {numeric: 'auto'});
    const deltaMs = unixMillis - Date.now();
    const absMs = Math.abs(deltaMs);
    for (const u of RELATIVE_UNITS) {
        if (absMs < u.limitMs) {
            return fmt.format(Math.round(deltaMs / u.divisor), u.unit);
        }
    }
    return fmt.format(Math.round(deltaMs / (365 * 86_400_000)), 'year');
}

export const TopPlaybooksList: React.FC<Props> = ({items, status, error}) => {
    if (status === 'loading') {
        return <TopPlaybooksSkeleton/>;
    }
    if (status === 'error') {
        return <p className='insights-card__error'>{error || 'Failed to load playbooks'}</p>;
    }
    if (items.length === 0) {
        return <WidgetEmptyState icon='product-playbooks'/>;
    }

    const top = items[0].num_runs || 1;

    const handleRowClick = (playbookId: string) => () => {
        trackInsightsEvent('open_playbook_from_top_playbooks_widget');
        navigateTo(`/playbooks/playbooks/${playbookId}`);
    };

    return (
        <div className='top-playbooks-container'>
            <div className='playbooks-list'>
                {items.map((playbook) => {
                    const barSize = (playbook.num_runs / top) * 100;
                    return (
                        <a
                            key={playbook.playbook_id}
                            className='playbook-item'
                            role='button'
                            tabIndex={0}
                            onClick={handleRowClick(playbook.playbook_id)}
                        >
                            <div className='display-info'>
                                <span className='display-name'>{playbook.title}</span>
                                <span className='last-run-time'>
                                    <FormattedMessage
                                        id='insights.topPlaybooks.lastRun'
                                        defaultMessage='Last run: {relativeTime}'
                                        values={{relativeTime: relativeTime(playbook.last_run_at)}}
                                    />
                                </span>
                            </div>
                            <div className='display-info run-info'>
                                <span
                                    className='horizontal-bar'
                                    style={{width: `${barSize}%`}}
                                />
                                <span className='last-run-time'>
                                    <FormattedMessage
                                        id='insights.topPlaybooks.totalRuns'
                                        defaultMessage='{total} runs'
                                        values={{total: playbook.num_runs}}
                                    />
                                </span>
                            </div>
                        </a>
                    );
                })}
            </div>
        </div>
    );
};
