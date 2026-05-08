// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_playbooks/top_playbooks_table/top_playbooks_table.tsx
// (commit 26617fcbdc).
//
// Adaptations:
//   - Data source is the Playbooks plugin's HTTP endpoints (the deprecated
//     code consumed `state.plugins.insightsHandlers.playbooks` from the
//     host's plugin registry, which no longer exists). The Playbooks
//     plugin server still serves the same shape at
//     `/plugins/playbooks/api/v0/playbooks/insights/...` (audited 2026-05-08).
//   - `Timestamp` (host-component) → `Intl.RelativeTimeFormat`.
//   - `useHistory` / `history.push` → `navigateTo` helper.

import React, {memo, useCallback, useMemo} from 'react';
import {FormattedMessage} from 'react-intl';

import {Client} from '../../../client/Client';
import type {TopPlaybook} from '../../../types';
import {navigateTo} from '../../../utils/navigation';
import {trackInsightsEvent} from '../../../utils/telemetry';
import type {Column, Row} from '../../DataGrid/DataGrid';
import {DataGrid} from '../../DataGrid/DataGrid';
import {usePaginatedTable} from '../usePaginatedTable';

import type {TableProps} from './types';

function relativeTime(unixMillis: number): string {
    if (!unixMillis) {
        return '';
    }
    const fmt = new Intl.RelativeTimeFormat(undefined, {numeric: 'auto'});
    const deltaMs = unixMillis - Date.now();
    const days = Math.round(deltaMs / 86_400_000);
    if (Math.abs(days) < 1) {
        return fmt.format(Math.round(deltaMs / 3_600_000), 'hour');
    }
    if (Math.abs(days) < 30) {
        return fmt.format(days, 'day');
    }
    if (Math.abs(days) < 365) {
        return fmt.format(Math.round(days / 30), 'month');
    }
    return fmt.format(Math.round(days / 365), 'year');
}

const TopPlaybooksTableComponent: React.FC<TableProps> = ({scope, timeRange, teamId}) => {
    const fetcher = useCallback((page: number, perPage: number) => {
        if (scope === 'team') {
            return Client.getTopPlaybooksForTeam(teamId, timeRange, {page, perPage});
        }
        return Client.getMyTopPlaybooks(timeRange, {page, perPage, teamId});
    }, [scope, teamId, timeRange]);

    const table = usePaginatedTable<TopPlaybook, {has_next: boolean; items: TopPlaybook[]}>(
        fetcher,
        [scope, teamId, timeRange],
    );

    const columns = useMemo<Column[]>(() => [
        {
            name: <FormattedMessage id='insights.topReactions.rank' defaultMessage='Rank'/>,
            field: 'rank',
            className: 'rankCell',
            width: 0.07,
        },
        {
            name: <FormattedMessage id='insights.topPlaybooksTable.playbook' defaultMessage='Playbook'/>,
            field: 'playbook',
            width: 0.4,
        },
        {
            name: <FormattedMessage id='insights.topPlaybooksTable.updates' defaultMessage='Last run'/>,
            field: 'lastRun',
            width: 0.23,
        },
        {
            name: <FormattedMessage id='insights.topPlaybooksTable.participants' defaultMessage='Total runs'/>,
            field: 'totalRuns',
            width: 0.3,
        },
    ], []);

    const rows = useMemo<Row[]>(() => {
        if (!table.items.length) {
            return [];
        }
        const top = table.items[0].num_runs || 1;
        return table.items.map((playbook, i) => {
            const barSize = playbook.num_runs / top;
            return {
                cells: {
                    rank: <span className='cell-text'>{table.page * table.perPage + i + 1}</span>,
                    playbook: (
                        <div className='channel-display-name'>
                            <span className='cell-text'>{playbook.title}</span>
                        </div>
                    ),
                    lastRun: <span className='cell-text'>{relativeTime(playbook.last_run_at)}</span>,
                    totalRuns: (
                        <div className='times-used-container'>
                            <span className='cell-text'>{playbook.num_runs}</span>
                            <span
                                className='horizontal-bar'
                                style={{flex: `${barSize} 0`}}
                            />
                        </div>
                    ),
                },
                onClick: () => {
                    trackInsightsEvent('open_playbook_from_top_playbooks_modal');
                    navigateTo(`/playbooks/playbooks/${playbook.playbook_id}`);
                },
            };
        });
    }, [table.items, table.page, table.perPage]);

    const startCount = table.page * table.perPage + 1;
    const endCount = startCount + table.items.length - 1;
    const total = table.hasNext ? endCount + 1 : endCount;

    return (
        <DataGrid
            columns={columns}
            rows={rows}
            loading={table.loading}
            nextPage={table.nextPage}
            previousPage={table.previousPage}
            startCount={startCount}
            endCount={endCount}
            total={Math.max(total, 0)}
            className='InsightsTable TopPlaybooksTable'
        />
    );
};

export const TopPlaybooksTable = memo(TopPlaybooksTableComponent);
