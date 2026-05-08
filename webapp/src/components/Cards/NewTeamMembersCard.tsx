import React, {useEffect} from 'react';
import {useDispatch, useSelector} from 'react-redux';

import {NewMembersTotal} from './NewMembersTotal';
import {NewTeamMembersList} from './NewTeamMembersList';

import {requestNewTeamMembers} from '../../redux/actions';
import {getCurrentTeamName} from '../../redux/mmSelectors';
import {selectNewTeamMembers} from '../../redux/selectors';
import type {TimeRange} from '../../types';
import {WidgetCard} from '../Card/widgetHoc';

interface Props {
    teamId: string;
    timeRange: TimeRange;
    onOpenDetails?: () => void;
}

export const NewTeamMembersCard: React.FC<Props> = ({teamId, timeRange, onOpenDetails}) => {
    const dispatch = useDispatch();
    const slice = useSelector((state: Record<string, unknown>) => selectNewTeamMembers(state, teamId, timeRange));
    const teamName = useSelector(getCurrentTeamName);

    useEffect(() => {
        if (!teamId) {
            return;
        }
        (dispatch as unknown as (a: unknown) => void)(requestNewTeamMembers(teamId, timeRange));
    }, [dispatch, teamId, timeRange]);

    return (
        <WidgetCard
            widgetType='newTeamMembers'
            scope='team'
            onOpenDetails={onOpenDetails}
        >
            <NewTeamMembersList
                items={slice.items}
                hasNext={slice.hasNext}
                status={slice.status}
                error={slice.error}
                teamName={teamName}
            />
            {slice.totalCount > 0 && onOpenDetails ? (
                <NewMembersTotal
                    total={slice.totalCount}
                    timeRange={timeRange}
                    openInsightsModal={onOpenDetails}
                />
            ) : null}
        </WidgetCard>
    );
};
