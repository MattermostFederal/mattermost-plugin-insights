// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_threads/top_threads_table/top_threads_table.tsx
// (commit 26617fcbdc).

import React, {memo, useCallback, useMemo} from 'react';
import {FormattedMessage} from 'react-intl';
import {useDispatch} from 'react-redux';

import type {TableProps} from './types';

import {Client} from '../../../client/Client';
import type {TopThread} from '../../../types';
import {openThreadRHS} from '../../../utils/navigation';
import {trackInsightsEvent} from '../../../utils/telemetry';
import type {Column, Row} from '../../DataGrid/DataGrid';
import {DataGrid} from '../../DataGrid/DataGrid';
import {usePaginatedTable} from '../usePaginatedTable';

function authorName(t: TopThread): string {
    const u = t.user_information;
    const fl = `${u.first_name} ${u.last_name}`.trim();
    return fl || u.username;
}

const TopThreadsTableComponent: React.FC<TableProps> = ({scope, timeRange, teamId}) => {
    const dispatch = useDispatch();

    const fetcher = useCallback((page: number, perPage: number) => {
        if (scope === 'team') {
            return Client.getTopThreadsForTeam(teamId, timeRange, {page, perPage});
        }
        return Client.getMyTopThreads(timeRange, {page, perPage, teamId: teamId || undefined});
    }, [scope, teamId, timeRange]);

    const table = usePaginatedTable<TopThread, {has_next: boolean; items: TopThread[]}>(
        fetcher,
        [scope, teamId, timeRange],
    );

    const columns = useMemo<Column[]>(() => [
        {
            name: <FormattedMessage
                id='insights.topThreads.thread'
                defaultMessage='Thread'
                  />,
            field: 'author',
        },
        {
            name: <FormattedMessage
                id='insights.topChannels.channel'
                defaultMessage='Channel'
                  />,
            field: 'channel',
        },
        {
            name: <FormattedMessage
                id='insights.topThreads.replies'
                defaultMessage='Replies'
                  />,
            field: 'replies',
        },
        {
            name: <FormattedMessage
                id='insights.topThreads.totalMessages'
                defaultMessage='Participants'
                  />,
            field: 'participants',
        },
    ], []);

    const rows = useMemo<Row[]>(() => table.items.map((thread) => {
        // `Post.reply_count` is the actual reply tally, hydrated by the
        // server from the host's `*model.Post`. `participants.length` is
        // a different metric (count of distinct posters in the thread).
        const replies = (thread.post as {reply_count?: number} | undefined)?.reply_count ?? 0;
        return {
            cells: {
                author: <span className='cell-text'>{authorName(thread)}</span>,
                channel: <span className='cell-text'>{thread.channel_display_name || thread.channel_name}</span>,
                replies: <span className='cell-text'>{replies}</span>,
                participants: <span className='cell-text'>{thread.participants.length}</span>,
            },
            onClick: () => {
                const postId = thread.post?.id;
                if (!postId) {
                    return;
                }
                trackInsightsEvent('open_thread_from_top_threads_modal');
                openThreadRHS(postId, dispatch as unknown as (a: unknown) => unknown);
            },
        };
    }), [table.items, dispatch]);

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
            className='InsightsTable TopThreadsTable'
        />
    );
};

export const TopThreadsTable = memo(TopThreadsTableComponent);
