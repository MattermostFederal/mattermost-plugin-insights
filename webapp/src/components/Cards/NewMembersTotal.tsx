// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_dms_and_new_members/new_members_total/new_members_total.tsx
// (commit 26617fcbdc).
//
// Adaptation: time-frame strings come from the same i18n keys
// (`insights.newMembers.today` / `lastSevenDays` / `lastTwentyEightDays`).

import React, {memo, useCallback} from 'react';
import {FormattedMessage} from 'react-intl';

import type {TimeRange} from '../../types';

interface Props {
    total: number;
    timeRange: TimeRange;
    openInsightsModal: () => void;
}

const NewMembersTotalComponent: React.FC<Props> = ({total, timeRange, openInsightsModal}) => {
    const timeFrameInfo = useCallback(() => {
        switch (timeRange) {
        case 'today':
            return (
                <FormattedMessage
                    id='insights.newMembers.today'
                    defaultMessage='Joined the team today'
                />
            );
        case '28_day':
            return (
                <FormattedMessage
                    id='insights.newMembers.lastTwentyEightDays'
                    defaultMessage='Joined the team in the last 28 days'
                />
            );
        case '7_day':
        default:
            return (
                <FormattedMessage
                    id='insights.newMembers.lastSevenDays'
                    defaultMessage='Joined the team in the last 7 days'
                />
            );
        }
    }, [timeRange]);

    return (
        <div className='new-members-item new-members-info'>
            <span className='total-count'>{total}</span>
            <div className='members-info'>
                <span className='time-range-info'>{timeFrameInfo()}</span>
                <button
                    className='see-all-button'
                    type='button'
                    onClick={(e) => {
                        e.stopPropagation();
                        openInsightsModal();
                    }}
                >
                    <FormattedMessage
                        id='insights.newMembers.seeAll'
                        defaultMessage='See all'
                    />
                    <i className='icon icon-chevron-right'/>
                </button>
            </div>
        </div>
    );
};

export const NewMembersTotal = memo(NewMembersTotalComponent);
