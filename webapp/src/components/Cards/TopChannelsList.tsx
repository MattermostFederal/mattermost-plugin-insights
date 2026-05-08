// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_channels/top_channels.tsx
// (commit 26617fcbdc^), specifically the per-row JSX inside the
// `.top-channel-list / .channel-row` block (lines 161-191 of the
// deprecated source).
//
// Adaptations:
//   - `<Link>` → row `onClick` → `navigateTo`
//   - `<OverlayTrigger><Tooltip>` → native `title` attribute on the count
//   - The deprecated `:nth-of-type(1..5)` SCSS gives each rank a unique
//     bar color (button-bg / online / away / dnd / new-message-separator).
//     Same tokens are ported into our `.top-channel-list .channel-row`
//     block in insights.scss.

import React, {memo} from 'react';
import {FormattedMessage, useIntl} from 'react-intl';

import type {Status} from '../../redux/types';
import type {TopChannel} from '../../types';
import {navigateTo} from '../../utils/navigation';
import {trackInsightsEvent} from '../../utils/telemetry';
import {WidgetEmptyState} from '../EmptyState/WidgetEmptyState';
import {TopChannelsSkeleton} from '../Skeleton/CardSkeletons';

interface Props {
    items: TopChannel[];
    hasNext: boolean;
    status: Status;
    error?: string;
    teamName?: string;
}

const TopChannelsListComponent: React.FC<Props> = ({items, status, error, teamName}) => {
    const intl = useIntl();

    if (status === 'loading') {
        return <TopChannelsSkeleton/>;
    }
    if (status === 'error') {
        return (
            <p className='insights-card__error'>{error || (
                <FormattedMessage
                    id='insights.topChannels.failed'
                    defaultMessage='Failed to load channels'
                />
            )}</p>
        );
    }
    if (items.length === 0) {
        return <WidgetEmptyState icon='globe'/>;
    }

    const top = items[0]?.message_count || 1;

    const handleRowClick = (channelName: string) => () => {
        if (!teamName) {
            return;
        }
        trackInsightsEvent('open_channel_from_top_channels_widget');
        navigateTo(`/${teamName}/channels/${channelName}`);
    };

    return (
        <div className='top-channel-list'>
            {items.map((channel) => {
                // Match the deprecated `barSize = (count / topCount) * 0.8`.
                const barSize = (channel.message_count / top) * 0.8;
                const icon = channel.type === 'P' ? 'icon-lock-outline' : 'icon-globe';
                const tooltip = intl.formatMessage(
                    {id: 'insights.topChannels.messageCount', defaultMessage: '{messageCount} total messages'},
                    {messageCount: channel.message_count},
                );
                return (
                    <a
                        key={channel.id}
                        className='channel-row'
                        role='button'
                        tabIndex={0}
                        onClick={handleRowClick(channel.name)}
                    >
                        <div className='channel-display-name'>
                            <span className='icon'>
                                <i className={`icon ${icon}`}/>
                            </span>
                            <span className='display-name'>{channel.display_name || channel.name}</span>
                        </div>
                        <div className='channel-message-count'>
                            <span
                                className='message-count'
                                title={tooltip}
                            >
                                {channel.message_count}
                            </span>
                            <span
                                className='horizontal-bar'
                                style={{flex: `${barSize} 0`}}
                            />
                        </div>
                    </a>
                );
            })}
        </div>
    );
};

export const TopChannelsList = memo(TopChannelsListComponent);
