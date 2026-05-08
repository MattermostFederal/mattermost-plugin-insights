// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_dms_and_new_members/top_dms_item/top_dms_item.tsx
// (commit 26617fcbdc^).
//
// Adaptations:
//   - `<Avatar url={imageURLForUser(...)}/>` (xl) → plugin `<HostAvatar/>`
//   - `<OverlayTrigger>` + `<Tooltip>` → native `title` attribute on the
//     count span. The host's tooltip styling is custom; for parity we
//     accept the slight visual difference.
//   - The horizontal bar's `flex: ${barSize} 0` is computed against the
//     top DM's post_count (matches deprecated math: `barSize = post_count
//     / topPostCount`).
//   - `<Link>` → `navigateTo` helper (host history push).
//   - `displayUsername(user, teammateNameDisplaySetting)` → simple
//     first/last/username preference. The deprecated host helper honors
//     a teammate-name-display preference (full name / nickname / username)
//     we don't have direct access to without bundling mattermost-redux;
//     the basic first-last-or-username choice matches the most common
//     server config.

import React, {memo} from 'react';
import {FormattedMessage, useIntl} from 'react-intl';

import type {Status} from '../../redux/types';
import type {TopDM} from '../../types';
import {navigateTo} from '../../utils/navigation';
import {trackInsightsEvent} from '../../utils/telemetry';
import {HostAvatar} from '../Avatar/HostAvatar';
import {WidgetEmptyState} from '../EmptyState/WidgetEmptyState';
import {TopDMsSkeleton} from '../Skeleton/CardSkeletons';

interface Props {
    items: TopDM[];
    hasNext: boolean;
    status: Status;
    error?: string;
    teamName?: string;
}

function partnerName(dm: TopDM): string {
    const u = dm.second_participant;
    const fl = `${u.first_name} ${u.last_name}`.trim();
    return fl || u.username;
}

const TopDMsListComponent: React.FC<Props> = ({items, status, error, teamName}) => {
    const intl = useIntl();

    if (status === 'loading') {
        return <TopDMsSkeleton/>;
    }
    if (status === 'error') {
        return (
            <p className='insights-card__error'>{error || (
                <FormattedMessage
                    id='insights.topDms.failed'
                    defaultMessage='Failed to load DMs'
                />
            )}</p>
        );
    }
    if (items.length === 0) {
        return <WidgetEmptyState icon='account-multiple-outline'/>;
    }

    const top = items[0]?.post_count || 1;

    const handleRowClick = (username: string) => () => {
        if (!teamName) {
            return;
        }
        trackInsightsEvent('open_dm_from_top_dms_widget');
        navigateTo(`/${teamName}/messages/@${username}`);
    };

    return (
        <div className='top-dms-container'>
            {items.map((dm) => {
                const barSize = dm.post_count / top;
                const tooltip = intl.formatMessage(
                    {id: 'insights.topChannels.messageCount', defaultMessage: '{messageCount} total messages'},
                    {messageCount: dm.post_count},
                );
                return (
                    <a
                        key={dm.second_participant.id}
                        className='top-dms-item'
                        role='button'
                        tabIndex={0}
                        onClick={handleRowClick(dm.second_participant.username)}
                    >
                        <HostAvatar
                            userID={dm.second_participant.id}
                            lastPictureUpdate={dm.second_participant.last_picture_update}
                            size='xl'
                        />
                        <div className='dm-info'>
                            <div className='dm-name'>{partnerName(dm)}</div>
                            {dm.second_participant.position ? (
                                <span className='dm-role'>{dm.second_participant.position}</span>
                            ) : null}
                            <div className='channel-message-count'>
                                <span
                                    className='message-count'
                                    title={tooltip}
                                >
                                    {dm.post_count}
                                </span>
                                <span
                                    className='horizontal-bar'
                                    style={{flex: `${barSize} 0`}}
                                />
                            </div>
                        </div>
                    </a>
                );
            })}
        </div>
    );
};

export const TopDMsList = memo(TopDMsListComponent);
