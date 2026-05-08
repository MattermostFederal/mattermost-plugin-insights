// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_dms_and_new_members/new_members_item/new_members_item.tsx
// (commit 26617fcbdc^).
//
// Adaptations:
//   - `<Avatar url={imageURLForUser(...)}/>` (xl) → plugin `<HostAvatar/>`
//   - `<RenderEmoji name='wave' size=14/>` → plugin emoji utility
//     pointing at `/static/emoji/<unified>.png` (the deprecated host
//     component pulled it through Redux + the same URL pattern)
//   - `<Link>` → row `onClick` → `navigateTo`
//   - `displayUsername(member, teammateNameDisplaySetting)` simplified to
//     first/last/username

import React, {memo} from 'react';
import {FormattedMessage} from 'react-intl';

import type {Status} from '../../redux/types';
import type {NewTeamMember} from '../../types';
import {getEmojiImageUrl} from '../../utils/emoji';
import {navigateTo} from '../../utils/navigation';
import {trackInsightsEvent} from '../../utils/telemetry';
import {HostAvatar} from '../Avatar/HostAvatar';
import {WidgetEmptyState} from '../EmptyState/WidgetEmptyState';
import {NewTeamMembersSkeleton} from '../Skeleton/CardSkeletons';

interface Props {
    items: NewTeamMember[];
    hasNext: boolean;
    status: Status;
    error?: string;
    teamName?: string;
}

function displayName(member: NewTeamMember): string {
    const fl = `${member.first_name} ${member.last_name}`.trim();
    return fl || member.username;
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

const NewTeamMembersListComponent: React.FC<Props> = ({items, status, error, teamName}) => {
    if (status === 'loading') {
        return <NewTeamMembersSkeleton/>;
    }
    if (status === 'error') {
        return (
            <p className='insights-card__error'>{error || (
                <FormattedMessage
                    id='insights.newMembers.failed'
                    defaultMessage='Failed to load new members'
                />
            )}</p>
        );
    }
    if (items.length === 0) {
        return <WidgetEmptyState icon='account-multiple-outline'/>;
    }

    const handleRowClick = (username: string) => () => {
        if (!teamName) {
            return;
        }
        trackInsightsEvent('open_new_members_from_new_members_widget');
        navigateTo(`/${teamName}/messages/@${username}`);
    };

    return (
        <div className='top-dms-container'>
            {items.map((member) => (
                <a
                    key={member.id}
                    className='top-dms-item new-members-item'
                    role='button'
                    tabIndex={0}
                    onClick={handleRowClick(member.username)}
                >
                    <HostAvatar
                        userID={member.id}
                        lastPictureUpdate={member.last_picture_update}
                        size='xl'
                    />
                    <div className='dm-info'>
                        <div className='dm-name'>{displayName(member)}</div>
                        {member.position ? (
                            <span className='dm-role'>{member.position}</span>
                        ) : null}
                        <div className='channel-message-count'>
                            <WaveEmoji/>
                            <div className='say-hello'>
                                <FormattedMessage
                                    id='insights.newMembers.sayHello'
                                    defaultMessage='Say hello'
                                />
                            </div>
                        </div>
                    </div>
                </a>
            ))}
        </div>
    );
};

export const NewTeamMembersList = memo(NewTeamMembersListComponent);
