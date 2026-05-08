import React, {useEffect} from 'react';
import {useDispatch, useSelector} from 'react-redux';

import {TopChannelsList} from './TopChannelsList';

import {requestTopChannels} from '../../redux/actions';
import {getCurrentTeamName} from '../../redux/mmSelectors';
import {selectChannelPostCountByDuration, selectTopChannels} from '../../redux/selectors';
import type {Scope, TimeRange} from '../../types';
import {WidgetCard} from '../Card/widgetHoc';
import {TopChannelsLineChart} from '../Charts/TopChannelsLineChart';

interface Props {
    scope: Scope;
    timeRange: TimeRange;
    teamId: string;
    onOpenDetails?: () => void;
}

const userScopeKey = (teamId: string) => teamId || 'global';

export const TopChannelsCard: React.FC<Props> = ({scope, timeRange, teamId, onOpenDetails}) => {
    const dispatch = useDispatch();
    const scopeKey = scope === 'team' ? teamId : userScopeKey(teamId);
    const slice = useSelector((state: Record<string, unknown>) => selectTopChannels(state, scope, scopeKey, timeRange));
    const postCountByDuration = useSelector(selectChannelPostCountByDuration);
    const teamName = useSelector(getCurrentTeamName);

    useEffect(() => {
        if (scope === 'team' && !teamId) {
            return;
        }
        (dispatch as unknown as (a: unknown) => void)(requestTopChannels(scope, scopeKey, timeRange, scope === 'my' ? teamId : undefined));
    }, [dispatch, scope, scopeKey, timeRange, teamId]);

    return (
        <WidgetCard
            widgetType='topChannels'
            scope={scope}
            onOpenDetails={onOpenDetails}
        >
            <TopChannelsLineChart
                topChannels={slice.items}
                postCountByDuration={postCountByDuration}
                timeRange={timeRange}
            />
            <TopChannelsList
                items={slice.items}
                hasNext={slice.hasNext}
                status={slice.status}
                error={slice.error}
                teamName={teamName}
            />
        </WidgetCard>
    );
};
