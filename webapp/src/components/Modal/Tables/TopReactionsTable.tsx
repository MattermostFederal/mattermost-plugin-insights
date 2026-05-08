// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_reactions/top_reactions_table/top_reactions_table.tsx
// (commit 26617fcbdc).

import React, {memo, useCallback, useEffect, useMemo} from 'react';
import {FormattedMessage} from 'react-intl';

import type {TableProps} from './types';

import {Client} from '../../../client/Client';
import type {TopReaction} from '../../../types';
import {getEmojiImageUrl, preloadEmojis} from '../../../utils/emoji';
import type {Column, Row} from '../../DataGrid/DataGrid';
import {DataGrid} from '../../DataGrid/DataGrid';
import {usePaginatedTable} from '../usePaginatedTable';

const ReactionEmoji: React.FC<{name: string; size: number}> = ({name, size}) => {
    const url = getEmojiImageUrl(name);
    const [errored, setErrored] = React.useState(false);
    if (!url || errored) {
        return <span>{`:${name}:`}</span>;
    }
    return (
        <img
            src={url}
            alt={`:${name}:`}
            width={size}
            height={size}
            className='emoticon'
            onError={() => setErrored(true)}
        />
    );
};

const TopReactionsTableComponent: React.FC<TableProps> = ({scope, timeRange, teamId}) => {
    const fetcher = useCallback((page: number, perPage: number) => {
        if (scope === 'team') {
            return Client.getTopReactionsForTeam(teamId, timeRange, {page, perPage});
        }
        return Client.getMyTopReactions(timeRange, {page, perPage, teamId: teamId || undefined});
    }, [scope, teamId, timeRange]);

    const table = usePaginatedTable<TopReaction, {has_next: boolean; items: TopReaction[]}>(
        fetcher,
        [scope, teamId, timeRange],
    );

    // Warm the browser cache for custom emojis (mirrors deprecated
    // `loadCustomEmojisIfNeeded` dispatch in `top_reactions_table.tsx`).
    useEffect(() => {
        if (table.items.length) {
            preloadEmojis(table.items.map((r) => r.emoji_name));
        }
    }, [table.items]);

    const columns = useMemo<Column[]>(() => [
        {
            name: <FormattedMessage
                id='insights.topReactions.rank'
                defaultMessage='Rank'
                  />,
            field: 'rank',
            className: 'rankCell',
            width: 0.2,
        },
        {
            name: <FormattedMessage
                id='insights.topReactions.reaction'
                defaultMessage='Reaction'
                  />,
            field: 'reaction',
        },
        {
            name: <FormattedMessage
                id='insights.topReactions.timesUsed'
                defaultMessage='Times used'
                  />,
            field: 'times_used',
        },
    ], []);

    const rows = useMemo<Row[]>(() => {
        if (!table.items.length) {
            return [];
        }
        const top = table.items[0].count || 1;
        return table.items.map((reaction, i) => {
            const barSize = reaction.count / top;
            return {
                cells: {
                    rank: <span className='cell-text'>{(table.page * table.perPage) + i + 1}</span>,
                    reaction: (
                        <div className='reaction-cell'>
                            <ReactionEmoji
                                name={reaction.emoji_name}
                                size={24}
                            />
                            <span className='cell-text'>{reaction.emoji_name}</span>
                        </div>
                    ),
                    times_used: (
                        <div className='times-used-container'>
                            <span className='cell-text'>{reaction.count}</span>
                            <span
                                className='horizontal-bar'
                                style={{flex: `${barSize} 0`}}
                            />
                        </div>
                    ),
                },
            };
        });
    }, [table.items, table.page, table.perPage]);

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
            className='InsightsTable TopReactionsTable'
        />
    );
};

export const TopReactionsTable = memo(TopReactionsTableComponent);
