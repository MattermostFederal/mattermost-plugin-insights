import React, {useEffect} from 'react';
import {useDispatch, useSelector} from 'react-redux';

import {TopDMsList} from './TopDMsList';

import {requestTopDMs} from '../../redux/actions';
import {getCurrentTeamName} from '../../redux/mmSelectors';
import {selectTopDMs} from '../../redux/selectors';
import type {TimeRange} from '../../types';
import {WidgetCard} from '../Card/widgetHoc';

interface Props {
    teamId: string;
    timeRange: TimeRange;
    onOpenDetails?: () => void;
}

const userScopeKey = (teamId: string) => teamId || 'global';

export const TopDMsCard: React.FC<Props> = ({teamId, timeRange, onOpenDetails}) => {
    const dispatch = useDispatch();
    const scopeKey = userScopeKey(teamId);
    const slice = useSelector((state: Record<string, unknown>) => selectTopDMs(state, scopeKey, timeRange));
    const teamName = useSelector(getCurrentTeamName);

    useEffect(() => {
        (dispatch as unknown as (a: unknown) => void)(requestTopDMs(scopeKey, timeRange));
    }, [dispatch, scopeKey, timeRange]);

    return (
        <WidgetCard
            widgetType='topDms'
            scope='my'
            onOpenDetails={onOpenDetails}
        >
            <TopDMsList
                items={slice.items}
                hasNext={slice.hasNext}
                status={slice.status}
                error={slice.error}
                teamName={teamName}
            />
        </WidgetCard>
    );
};
