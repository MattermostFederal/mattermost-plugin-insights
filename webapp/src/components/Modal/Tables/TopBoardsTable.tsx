// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_boards/top_boards_table/top_boards_table.tsx
// (commit 26617fcbdc).
//
// Adaptations:
//   - Data source is this plugin's `/users/me/top/boards` and
//     `/teams/{team_id}/top/boards` routes (the deprecated code consumed
//     `state.plugins.insightsHandlers.focalboard`, which the host removed
//     alongside Insights, and the Boards plugin removed its insights
//     server in commit c8e729b6, June 2024).
//   - `<Avatars/>` (host-component) → small inline avatar pills, same
//     shape as TopBoardsList.
//   - `useHistory` / `history.push` → `navigateTo` helper.

import React, {memo, useCallback, useMemo} from 'react';
import {FormattedMessage} from 'react-intl';

import type {TableProps} from './types';

import {Client} from '../../../client/Client';
import type {TopBoard} from '../../../types';
import {navigateTo} from '../../../utils/navigation';
import {trackInsightsEvent} from '../../../utils/telemetry';
import type {Column, Row} from '../../DataGrid/DataGrid';
import {DataGrid} from '../../DataGrid/DataGrid';
import {usePaginatedTable} from '../usePaginatedTable';

interface MMComponents {
    imageURLForUser?: (userID: string, lastPictureUpdate?: number) => string;
}

function avatarURL(userID: string): string {
    const w = window as unknown as {Components?: MMComponents};
    const helper = w.Components?.imageURLForUser;
    if (helper) {
        return helper(userID);
    }
    return `/api/v4/users/${userID}/image`;
}

function activeUserIDs(b: TopBoard): string[] {
    const u = b.activeUsers as unknown;
    if (typeof u === 'string') {
        return u ? u.split(',') : [];
    }
    return Array.isArray(u) ? u : [];
}

const MAX_AVATARS = 4;

const ParticipantAvatars: React.FC<{userIDs: string[]}> = ({userIDs}) => {
    const visible = userIDs.slice(0, MAX_AVATARS);
    const overflow = userIDs.length - visible.length;
    return (
        <div className='Avatars Avatars___xs'>
            {visible.map((id) => (
                <img
                    key={id}
                    className='Avatar Avatar___xs'
                    src={avatarURL(id)}
                    alt=''
                />
            ))}
            {overflow > 0 ? (
                <span className='Avatars__overflow'>{`+${overflow}`}</span>
            ) : null}
        </div>
    );
};

const TopBoardsTableComponent: React.FC<TableProps> = ({scope, timeRange, teamId}) => {
    const fetcher = useCallback((page: number, perPage: number) => {
        if (scope === 'team') {
            return Client.getTopBoardsForTeam(teamId, timeRange, {page, perPage});
        }
        return Client.getMyTopBoards(timeRange, {page, perPage, teamId});
    }, [scope, teamId, timeRange]);

    const table = usePaginatedTable<TopBoard, {has_next: boolean; items: TopBoard[]}>(
        fetcher,
        [scope, teamId, timeRange],
    );

    const columns = useMemo<Column[]>(() => [
        {
            name: <FormattedMessage
                id='insights.topReactions.rank'
                defaultMessage='Rank'
                  />,
            field: 'rank',
            className: 'rankCell',
            width: 0.07,
        },
        {
            name: <FormattedMessage
                id='insights.topBoardsTable.board'
                defaultMessage='Board'
                  />,
            field: 'board',
            width: 0.7,
        },
        {
            name: <FormattedMessage
                id='insights.topBoardsTable.updates'
                defaultMessage='Updates'
                  />,
            field: 'updates',
            width: 0.08,
        },
        {
            name: <FormattedMessage
                id='insights.topBoardsTable.participants'
                defaultMessage='Participants'
                  />,
            field: 'participants',
            width: 0.15,
        },
    ], []);

    const rows = useMemo<Row[]>(() => table.items.map((board, i) => ({
        cells: {
            rank: <span className='cell-text'>{(table.page * table.perPage) + i + 1}</span>,
            board: (
                <div className='board-item'>
                    <span className='board-icon'>{board.icon}</span>
                    <span className='board-title'>{board.title}</span>
                </div>
            ),
            updates: <span className='board-updates'>{board.activityCount}</span>,
            participants: <ParticipantAvatars userIDs={activeUserIDs(board)}/>,
        },
        onClick: () => {
            trackInsightsEvent('open_board_from_top_boards_modal');
            navigateTo(`/boards/team/${teamId}/${board.boardID}`);
        },
    })), [table.items, table.page, table.perPage, teamId]);

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
            className='InsightsTable TopBoardsTable'
        />
    );
};

export const TopBoardsTable = memo(TopBoardsTableComponent);
