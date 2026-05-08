import React, {useEffect} from 'react';
import {useDispatch, useSelector} from 'react-redux';

import {TopInactiveChannelsList} from './TopInactiveChannelsList';

import {requestTopInactiveChannels} from '../../redux/actions';
import {getCurrentTeamName} from '../../redux/mmSelectors';
import {selectTopInactiveChannels} from '../../redux/selectors';
import type {Scope, TimeRange} from '../../types';
import {WidgetCard} from '../Card/widgetHoc';

interface Props {
    scope: Scope;
    timeRange: TimeRange;
    teamId: string;
    onOpenDetails?: () => void;
}

const userScopeKey = (teamId: string) => teamId || 'global';

export const TopInactiveChannelsCard: React.FC<Props> = ({scope, timeRange, teamId, onOpenDetails}) => {
    const dispatch = useDispatch();
    const scopeKey = scope === 'team' ? teamId : userScopeKey(teamId);
    const slice = useSelector((state: Record<string, unknown>) => selectTopInactiveChannels(state, scope, scopeKey, timeRange));
    const teamName = useSelector(getCurrentTeamName);

    useEffect(() => {
        if (scope === 'team' && !teamId) {
            return;
        }
        (dispatch as unknown as (a: unknown) => void)(requestTopInactiveChannels(scope, scopeKey, timeRange, scope === 'my' ? teamId : undefined));
    }, [dispatch, scope, scopeKey, timeRange, teamId]);

    return (
        <WidgetCard
            widgetType='topInactiveChannels'
            scope={scope}
            onOpenDetails={onOpenDetails}
        >
            <TopInactiveChannelsList
                items={slice.items}
                hasNext={slice.hasNext}
                status={slice.status}
                error={slice.error}
                teamName={teamName}
            />
        </WidgetCard>
    );
};
