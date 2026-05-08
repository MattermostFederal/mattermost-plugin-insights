// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/least_active_channels/least_active_channels_item/least_active_channels_item.tsx
// (commit 26617fcbdc^).
//
// Adaptations:
//   - `<Avatars userIds={participants}/>` → plugin `<Avatars/>`.
//   - `<Timestamp ... useTime={false} units={[...]}>` → plugin
//     `<RelativeTimestamp/>` which uses the host `Timestamp` when
//     available and falls back to `Intl.RelativeTimeFormat`.
//   - `<Link>` → row `onClick` → `navigateTo`.
//   - `<ChannelActionsMenu/>` already ported.

import React, {memo} from 'react';
import {FormattedMessage} from 'react-intl';
import {useSelector} from 'react-redux';

import {getCurrentUserId, isMemberOfChannel} from '../../redux/mmSelectors';
import type {Status} from '../../redux/types';
import type {TopInactiveChannel} from '../../types';
import {navigateTo} from '../../utils/navigation';
import {trackInsightsEvent} from '../../utils/telemetry';
import {Avatars} from '../Avatar/Avatars';
import {ChannelActionsMenu} from '../ChannelActions/ChannelActionsMenu';
import {WidgetEmptyState} from '../EmptyState/WidgetEmptyState';
import {TopInactiveChannelsSkeleton} from '../Skeleton/CardSkeletons';
import {RelativeTimestamp} from '../Timestamp/RelativeTimestamp';

interface Props {
    items: TopInactiveChannel[];
    hasNext: boolean;
    status: Status;
    error?: string;
    teamName?: string;
}

const ChannelActionsCell: React.FC<{channel: TopInactiveChannel; teamName: string}> = ({channel, teamName}) => {
    const userID = useSelector(getCurrentUserId);
    const isMember = useSelector((state: unknown) => isMemberOfChannel(state, channel.id));
    if (!userID) {
        return null;
    }
    return (
        <ChannelActionsMenu
            channel={channel}
            teamName={teamName}
            currentUserId={userID}
            isMember={isMember}
        />
    );
};

const TopInactiveChannelsListComponent: React.FC<Props> = ({items, status, error, teamName}) => {
    if (status === 'loading') {
        return <TopInactiveChannelsSkeleton/>;
    }
    if (status === 'error') {
        return (
            <p className='insights-card__error'>{error || (
                <FormattedMessage
                    id='insights.leastActiveChannels.failed'
                    defaultMessage='Failed to load channels'
                />
            )}</p>
        );
    }
    if (items.length === 0) {
        return <WidgetEmptyState icon='globe'/>;
    }

    const handleRowClick = (channelName: string) => () => {
        if (!teamName) {
            return;
        }
        trackInsightsEvent('open_channel_from_least_active_channels_widget');
        navigateTo(`/${teamName}/channels/${channelName}`);
    };

    return (
        <div className='channel-list'>
            {items.map((item) => {
                const icon = item.type === 'P' ? 'icon-lock-outline' : 'icon-globe';
                return (
                    <a
                        key={item.id}
                        className='channel-row'
                        role='button'
                        tabIndex={0}
                        onClick={handleRowClick(item.name)}
                    >
                        <div className='channel-info'>
                            <div className='channel-display-name'>
                                <span className='icon'>
                                    <i className={`icon ${icon}`}/>
                                </span>
                                <span className='display-name'>{item.display_name || item.name}</span>
                            </div>
                            <span className='last-activity'>
                                {item.last_activity_at === 0 ? (
                                    <FormattedMessage
                                        id='insights.leastActiveChannels.lastActivityNone'
                                        defaultMessage='No activity'
                                    />
                                ) : (
                                    <FormattedMessage
                                        id='insights.leastActiveChannels.lastActivity'
                                        defaultMessage='Last activity: {time}'
                                        values={{
                                            time: (
                                                <RelativeTimestamp
                                                    value={item.last_activity_at}
                                                    units={['now', 'minute', 'hour', 'day', 'week', 'month']}
                                                    useTime={false}
                                                />
                                            ),
                                        }}
                                    />
                                )}
                            </span>
                        </div>
                        <Avatars
                            userIDs={item.participants}
                            size='xs'
                        />
                        {teamName ? (
                            <ChannelActionsCell
                                channel={item}
                                teamName={teamName}
                            />
                        ) : null}
                    </a>
                );
            })}
        </div>
    );
};

export const TopInactiveChannelsList = memo(TopInactiveChannelsListComponent);
