// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_dms_and_new_members/new_members_table/new_members_table.tsx
// (commit 26617fcbdc).
//
// Adaptation: the deprecated modal listed bare names. This one carries the
// avatar and the "Say hello" wave the card used to show, because the card is
// gone — the tile in the governance summary row opens this modal instead, so
// it has to be the place you actually recognise people.

import React, {memo, useCallback, useMemo, useState} from 'react';
import {FormattedMessage} from 'react-intl';
import {useSelector} from 'react-redux';

import type {TableProps} from './types';

import {Client} from '../../../client/Client';
import {getCurrentTeamName} from '../../../redux/mmSelectors';
import type {NewTeamMember, NewTeamMembersResponse} from '../../../types';
import {getEmojiImageUrl} from '../../../utils/emoji';
import {navigateTo} from '../../../utils/navigation';
import {trackInsightsEvent} from '../../../utils/telemetry';
import {HostAvatar} from '../../Avatar/HostAvatar';
import type {Column, Row} from '../../DataGrid/DataGrid';
import {DataGrid} from '../../DataGrid/DataGrid';
import {usePaginatedTable} from '../usePaginatedTable';

function displayName(m: NewTeamMember): string {
    const fl = `${m.first_name} ${m.last_name}`.trim();
    return fl || m.username;
}

function relativeTime(unixMillis: number): string {
    if (!unixMillis) {
        return '';
    }
    const days = Math.floor((Date.now() - unixMillis) / 86_400_000);
    if (days < 1) {
        return 'today';
    }
    if (days === 1) {
        return 'yesterday';
    }
    return `${days} days ago`;
}

const WaveEmoji: React.FC = () => {
    const url = getEmojiImageUrl('wave');
    if (!url) {
        return <span aria-hidden='true'>{'👋'}</span>;
    }
    return (
        <img
            src={url}
            alt=''
            width={14}
            height={14}
            className='emoticon'
        />
    );
};

const NewMembersTableComponent: React.FC<TableProps> = ({timeRange, teamId}) => {
    const [totalCount, setTotalCount] = useState(0);
    const teamName = useSelector(getCurrentTeamName);

    const fetcher = useCallback((page: number, perPage: number) =>
        Client.getNewTeamMembers(teamId, timeRange, {page, perPage}), [teamId, timeRange]);

    const onResponse = useCallback((resp: NewTeamMembersResponse) => {
        setTotalCount(resp.total_count ?? 0);
    }, []);

    const table = usePaginatedTable<NewTeamMember, NewTeamMembersResponse>(
        fetcher,
        [teamId, timeRange],
        onResponse,
    );

    const columns = useMemo<Column[]>(() => [
        {
            name: <FormattedMessage
                id='insights.newMembers.member'
                defaultMessage='Team member'
                  />,
            field: 'name',
        },
        {
            name: <FormattedMessage
                id='insights.newMembers.position'
                defaultMessage='Position'
                  />,
            field: 'position',
        },
        {
            name: <FormattedMessage
                id='insights.newMembers.joined'
                defaultMessage='Date joined'
                  />,
            field: 'joined',
        },
        {
            name: '',
            field: 'action',
            width: 0.4,
        },
    ], []);

    const rows = useMemo<Row[]>(() => table.items.map((member) => ({
        cells: {
            name: (
                <div className='new-members-cell'>
                    <HostAvatar
                        userID={member.id}
                        lastPictureUpdate={member.last_picture_update}
                        size='md'
                    />
                    <div className='new-members-cell__names'>
                        <span className='cell-text'>{displayName(member)}</span>
                        <span className='new-members-cell__username'>{`@${member.username}`}</span>
                    </div>
                </div>
            ),
            position: <span className='cell-text'>{member.position || ''}</span>,
            joined: <span className='cell-text'>{relativeTime(member.create_at)}</span>,
            action: (
                <span className='new-members-cell__hello'>
                    <WaveEmoji/>
                    <FormattedMessage
                        id='insights.newMembers.sayHello'
                        defaultMessage='Say hello'
                    />
                </span>
            ),
        },
        onClick: teamName ? () => {
            trackInsightsEvent('open_new_members_from_new_members_modal');
            navigateTo(`/${teamName}/messages/@${member.username}`);
        } : undefined,
    })), [table.items, teamName]);

    const startCount = (table.page * table.perPage) + 1;
    const endCount = (startCount + table.items.length) - 1;

    // Prefer the server-supplied `total_count` over the inferred has-next
    // boundary; the New Team Members route returns it.
    const total = totalCount || (table.hasNext ? endCount + 1 : endCount);

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
            className='InsightsTable NewMembersTable'
        />
    );
};

export const NewMembersTable = memo(NewMembersTableComponent);
