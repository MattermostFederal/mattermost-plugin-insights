// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_boards/top_boards.tsx
// (commit 26617fcbdc).
//
// Adaptations:
//   - Skeleton blocks moved into the shared `TopBoardsSkeleton`.
//   - Row click goes through the plugin's `navigateTo` helper instead of
//     a `<Link to=...>` so the host's history is reused without pulling
//     in `react-router-dom`.
//   - `<Avatars/>` (host-component) → a small CSS pill listing the first
//     few participant ids and a "+N" overflow indicator. The deprecated
//     code rendered avatar images via the host's avatar URL helper; the
//     plugin can reuse `imageURLForUser` exposed at
//     `window.Components.imageURLForUser` when available, falling back to
//     `/api/v4/users/{id}/image`.
//   - MM-49023: deprecated payloads sometimes carried `activeUsers` as a
//     comma-joined string. Both shapes are normalized here.

import React from 'react';
import {FormattedMessage} from 'react-intl';

import type {Status} from '../../redux/types';
import type {TopBoard} from '../../types';
import {navigateTo} from '../../utils/navigation';
import {trackInsightsEvent} from '../../utils/telemetry';
import {WidgetEmptyState} from '../EmptyState/WidgetEmptyState';
import {TopBoardsSkeleton} from '../Skeleton/CardSkeletons';

interface Props {
    items: TopBoard[];
    hasNext: boolean;
    status: Status;
    error?: string;
    teamId?: string;
}

function activeUserIDs(b: TopBoard): string[] {
    const u = b.activeUsers as unknown;
    if (typeof u === 'string') {
        return u ? u.split(',') : [];
    }
    return Array.isArray(u) ? u : [];
}

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

const MAX_AVATARS = 4;

const ActiveUserAvatars: React.FC<{userIDs: string[]}> = ({userIDs}) => {
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

export const TopBoardsList: React.FC<Props> = ({items, status, error, teamId}) => {
    if (status === 'loading') {
        return <TopBoardsSkeleton/>;
    }
    if (status === 'error') {
        return <p className='insights-card__error'>{error || 'Failed to load boards'}</p>;
    }
    if (items.length === 0) {
        return <WidgetEmptyState icon='product-boards'/>;
    }

    const handleRowClick = (boardID: string) => () => {
        if (!teamId) {
            return;
        }
        trackInsightsEvent('open_board_from_top_boards_widget');
        navigateTo(`/boards/team/${teamId}/${boardID}`);
    };

    return (
        <div className='top-board-container'>
            <div className='board-list'>
                {items.map((board) => (
                    <a
                        key={board.boardID}
                        className='board-item'
                        role='button'
                        tabIndex={0}
                        onClick={handleRowClick(board.boardID)}
                    >
                        <span className='board-icon'>{board.icon}</span>
                        <div className='display-info'>
                            <span className='display-name'>{board.title}</span>
                            <span className='update-counts'>
                                <FormattedMessage
                                    id='insights.topBoards.updates'
                                    defaultMessage='{updateCount} updates'
                                    values={{updateCount: board.activityCount}}
                                />
                            </span>
                        </div>
                        <ActiveUserAvatars userIDs={activeUserIDs(board)}/>
                    </a>
                ))}
            </div>
        </div>
    );
};
