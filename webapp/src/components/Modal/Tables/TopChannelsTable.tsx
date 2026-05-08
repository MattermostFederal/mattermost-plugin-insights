// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_channels/top_channels_table/top_channels_table.tsx
// (commit 26617fcbdc).

import React, {memo, useCallback, useMemo} from 'react';
import {FormattedMessage} from 'react-intl';
import {useSelector} from 'react-redux';

import {Client} from '../../../client/Client';
import {getCurrentTeamName} from '../../../redux/mmSelectors';
import type {TopChannel} from '../../../types';
import {navigateTo} from '../../../utils/navigation';
import {trackInsightsEvent} from '../../../utils/telemetry';
import type {Column, Row} from '../../DataGrid/DataGrid';
import {DataGrid} from '../../DataGrid/DataGrid';
import {usePaginatedTable} from '../usePaginatedTable';

import type {TableProps} from './types';

const TopChannelsTableComponent: React.FC<TableProps> = ({scope, timeRange, teamId}) => {
    const teamName = useSelector(getCurrentTeamName);

    const fetcher = useCallback((page: number, perPage: number) => {
        if (scope === 'team') {
            return Client.getTopChannelsForTeam(teamId, timeRange, {page, perPage});
        }
        return Client.getMyTopChannels(timeRange, {page, perPage, teamId: teamId || undefined});
    }, [scope, teamId, timeRange]);

    const table = usePaginatedTable<TopChannel, {has_next: boolean; items: TopChannel[]}>(
        fetcher,
        [scope, teamId, timeRange],
    );

    const columns = useMemo<Column[]>(() => [
        {
            name: <FormattedMessage id='insights.topReactions.rank' defaultMessage='Rank'/>,
            field: 'rank',
            className: 'rankCell',
            width: 0.2,
        },
        {
            name: <FormattedMessage id='insights.topChannels.channel' defaultMessage='Channel'/>,
            field: 'channel',
        },
        {
            name: <FormattedMessage id='insights.topChannels.totalMessages' defaultMessage='Total messages'/>,
            field: 'messages',
        },
    ], []);

    const rows = useMemo<Row[]>(() => {
        if (!table.items.length) {
            return [];
        }
        const top = table.items[0].message_count || 1;
        return table.items.map((channel, i) => {
            const barSize = channel.message_count / top;
            const icon = channel.type === 'P' ? 'lock-outline' : 'globe';
            return {
                cells: {
                    rank: <span className='cell-text'>{table.page * table.perPage + i + 1}</span>,
                    channel: (
                        <div className='channel-cell'>
                            <i className={`icon icon-${icon}`}/>
                            <span className='cell-text'>{channel.display_name || channel.name}</span>
                        </div>
                    ),
                    messages: (
                        <div className='times-used-container'>
                            <span className='cell-text'>{channel.message_count}</span>
                            <span
                                className='horizontal-bar'
                                style={{flex: `${barSize} 0`}}
                            />
                        </div>
                    ),
                },
                onClick: teamName ? () => {
                    trackInsightsEvent('open_channel_from_top_channels_modal');
                    navigateTo(`/${teamName}/channels/${channel.name}`);
                } : undefined,
            };
        });
    }, [table.items, table.page, table.perPage, teamName]);

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
            className='InsightsTable TopChannelsTable'
        />
    );
};

export const TopChannelsTable = memo(TopChannelsTableComponent);
