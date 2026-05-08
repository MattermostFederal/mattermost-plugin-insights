// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_dms_and_new_members/top_dms_table/top_dms_table.tsx
// (commit 26617fcbdc^).
//
// Columns match the deprecated table verbatim:
//   Rank | User (avatar + display name) | Sent | Received | Total messages
//
// Adaptations:
//   - `<Avatar>` → plugin `<HostAvatar/>`.
//   - `<Link to={dmUrl}>` → row `onClick` → `navigateTo`.
//   - `displayUsername(user, teammateNameDisplaySetting)` simplified to
//     first/last/username.

import React, {memo, useCallback, useMemo} from 'react';
import {FormattedMessage} from 'react-intl';
import {useSelector} from 'react-redux';

import type {TableProps} from './types';

import {Client} from '../../../client/Client';
import {getCurrentTeamName} from '../../../redux/mmSelectors';
import type {TopDM} from '../../../types';
import {navigateTo} from '../../../utils/navigation';
import {trackInsightsEvent} from '../../../utils/telemetry';
import {HostAvatar} from '../../Avatar/HostAvatar';
import type {Column, Row} from '../../DataGrid/DataGrid';
import {DataGrid} from '../../DataGrid/DataGrid';
import {usePaginatedTable} from '../usePaginatedTable';

function partnerName(d: TopDM): string {
    const u = d.second_participant;
    const fl = `${u.first_name} ${u.last_name}`.trim();
    return fl || u.username;
}

const TopDMsTableComponent: React.FC<TableProps> = ({timeRange}) => {
    const teamName = useSelector(getCurrentTeamName);

    const fetcher = useCallback((page: number, perPage: number) =>
        Client.getMyTopDMs(timeRange, {page, perPage}), [timeRange]);

    const table = usePaginatedTable<TopDM, {has_next: boolean; items: TopDM[]}>(
        fetcher,
        [timeRange],
    );

    const columns = useMemo<Column[]>(() => [
        {
            name: <FormattedMessage
                id='insights.topReactions.rank'
                defaultMessage='Rank'
                  />,
            field: 'rank',
            className: 'rankCell',
            width: 0.05,
        },
        {
            name: <FormattedMessage
                id='insights.topDMs.user'
                defaultMessage='User'
                  />,
            field: 'user',
            width: 0.4,
        },
        {
            name: <FormattedMessage
                id='insights.topDMs.sentMessages'
                defaultMessage='Sent'
                  />,
            field: 'sent',
            className: 'message-count',
            width: 0.15,
        },
        {
            name: <FormattedMessage
                id='insights.topDMs.receivedMessages'
                defaultMessage='Received'
                  />,
            field: 'received',
            className: 'message-count',
            width: 0.15,
        },
        {
            name: <FormattedMessage
                id='insights.topDMs.totalMessages'
                defaultMessage='Total messages'
                  />,
            field: 'total',
            width: 0.25,
        },
    ], []);

    const rows = useMemo<Row[]>(() => table.items.map((dm, i) => ({
        cells: {
            rank: <span className='cell-text'>{(table.page * table.perPage) + i + 1}</span>,
            user: (
                <div className='user-info'>
                    <HostAvatar
                        userID={dm.second_participant.id}
                        lastPictureUpdate={dm.second_participant.last_picture_update}
                        size='sm'
                    />
                    <span className='display-name'>{partnerName(dm)}</span>
                </div>
            ),
            sent: <span className='cell-text'>{dm.outgoing_message_count}</span>,
            received: <span className='cell-text'>{dm.post_count - dm.outgoing_message_count}</span>,
            total: <span className='cell-text'>{dm.post_count}</span>,
        },
        onClick: teamName ? () => {
            trackInsightsEvent('open_dm_from_top_dms_modal');
            navigateTo(`/${teamName}/messages/@${dm.second_participant.username}`);
        } : undefined,
    })), [table.items, table.page, table.perPage, teamName]);

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
            className='InsightsTable TopDMsTable'
        />
    );
};

export const TopDMsTable = memo(TopDMsTableComponent);
