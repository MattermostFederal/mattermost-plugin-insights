// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/least_active_channels/least_active_channels_table/least_active_channels_table.tsx
// (commit 26617fcbdc).

import React, {memo, useCallback, useMemo} from 'react';
import {FormattedMessage} from 'react-intl';
import {useSelector} from 'react-redux';

import type {TableProps} from './types';

import {Client} from '../../../client/Client';
import {getCurrentTeamName} from '../../../redux/mmSelectors';
import type {TopInactiveChannel} from '../../../types';
import {navigateTo} from '../../../utils/navigation';
import {trackInsightsEvent} from '../../../utils/telemetry';
import type {Column, Row} from '../../DataGrid/DataGrid';
import {DataGrid} from '../../DataGrid/DataGrid';
import {usePaginatedTable} from '../usePaginatedTable';

function formatLastActivity(unixMillis: number): string {
    if (unixMillis <= 0) {
        return 'Never';
    }
    return new Date(unixMillis).toLocaleDateString();
}

const LeastActiveChannelsTableComponent: React.FC<TableProps> = ({scope, timeRange, teamId}) => {
    const teamName = useSelector(getCurrentTeamName);

    const fetcher = useCallback((page: number, perPage: number) => {
        if (scope === 'team') {
            return Client.getTopInactiveChannelsForTeam(teamId, timeRange, {page, perPage});
        }
        return Client.getMyTopInactiveChannels(timeRange, {page, perPage, teamId: teamId || undefined});
    }, [scope, teamId, timeRange]);

    const table = usePaginatedTable<TopInactiveChannel, {has_next: boolean; items: TopInactiveChannel[]}>(
        fetcher,
        [scope, teamId, timeRange],
    );

    const columns = useMemo<Column[]>(() => [
        {
            name: <FormattedMessage
                id='insights.leastActiveChannels.channel'
                defaultMessage='Channel'
                  />,
            field: 'channel',
        },
        {
            name: <FormattedMessage
                id='insights.leastActiveChannels.members'
                defaultMessage='Members'
                  />,
            field: 'members',
        },
        {
            name: <FormattedMessage
                id='insights.leastActiveChannels.lastActivityCell'
                defaultMessage='Last activity'
                  />,
            field: 'last_activity',
        },
    ], []);

    const rows = useMemo<Row[]>(() => table.items.map((channel) => {
        const icon = channel.type === 'P' ? 'lock-outline' : 'globe';
        return {
            cells: {
                channel: (
                    <div className='channel-cell'>
                        <i className={`icon icon-${icon}`}/>
                        <span className='cell-text'>{channel.display_name || channel.name}</span>
                    </div>
                ),
                members: <span className='cell-text'>{channel.participants.length}</span>,
                last_activity: <span className='cell-text'>{formatLastActivity(channel.last_activity_at)}</span>,
            },
            onClick: teamName ? () => {
                trackInsightsEvent('open_channel_from_least_active_channels_modal');
                navigateTo(`/${teamName}/channels/${channel.name}`);
            } : undefined,
        };
    }), [table.items, teamName]);

    const startCount = (table.page * table.perPage) + 1;
    const endCount = (startCount + table.items.length) - 1;
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
            className='InsightsTable LeastActiveChannelsTable'
        />
    );
};

export const LeastActiveChannelsTable = memo(LeastActiveChannelsTableComponent);
