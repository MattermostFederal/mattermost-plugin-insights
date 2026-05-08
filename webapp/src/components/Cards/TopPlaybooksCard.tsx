import React, {useEffect} from 'react';
import {useDispatch, useSelector} from 'react-redux';

import {TopPlaybooksList} from './TopPlaybooksList';

import {requestTopPlaybooks} from '../../redux/actions';
import {selectTopPlaybooks} from '../../redux/selectors';
import type {Scope, TimeRange} from '../../types';
import {WidgetCard} from '../Card/widgetHoc';

interface Props {
    scope: Scope;
    timeRange: TimeRange;
    teamId: string;
    onOpenDetails?: () => void;
}

const userScopeKey = (teamId: string) => teamId || 'global';

export const TopPlaybooksCard: React.FC<Props> = ({scope, timeRange, teamId, onOpenDetails}) => {
    const dispatch = useDispatch();
    const scopeKey = scope === 'team' ? teamId : userScopeKey(teamId);
    const slice = useSelector((state: Record<string, unknown>) => selectTopPlaybooks(state, scope, scopeKey, timeRange));

    useEffect(() => {
        if (!teamId) {
            return;
        }
        (dispatch as unknown as (a: unknown) => void)(requestTopPlaybooks(scope, scopeKey, timeRange, teamId));
    }, [dispatch, scope, scopeKey, timeRange, teamId]);

    return (
        <WidgetCard
            widgetType='topPlaybooks'
            scope={scope}
            onOpenDetails={onOpenDetails}
        >
            <TopPlaybooksList
                items={slice.items}
                hasNext={slice.hasNext}
                status={slice.status}
                error={slice.error}
            />
        </WidgetCard>
    );
};
