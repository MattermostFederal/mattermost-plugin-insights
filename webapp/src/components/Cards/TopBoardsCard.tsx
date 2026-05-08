import React, {useEffect} from 'react';
import {useDispatch, useSelector} from 'react-redux';

import {requestTopBoards} from '../../redux/actions';
import {selectTopBoards} from '../../redux/selectors';
import type {Scope, TimeRange} from '../../types';
import {WidgetCard} from '../Card/widgetHoc';

import {TopBoardsList} from './TopBoardsList';

interface Props {
    scope: Scope;
    timeRange: TimeRange;
    teamId: string;
    onOpenDetails?: () => void;
}

const userScopeKey = (teamId: string) => teamId || 'global';

export const TopBoardsCard: React.FC<Props> = ({scope, timeRange, teamId, onOpenDetails}) => {
    const dispatch = useDispatch();
    const scopeKey = scope === 'team' ? teamId : userScopeKey(teamId);
    const slice = useSelector((state: Record<string, unknown>) => selectTopBoards(state, scope, scopeKey, timeRange));

    useEffect(() => {
        if (!teamId) {
            return;
        }
        (dispatch as unknown as (a: unknown) => void)(requestTopBoards(scope, scopeKey, timeRange, teamId));
    }, [dispatch, scope, scopeKey, timeRange, teamId]);

    return (
        <WidgetCard
            widgetType='topBoards'
            scope={scope}
            onOpenDetails={onOpenDetails}
        >
            <TopBoardsList
                items={slice.items}
                hasNext={slice.hasNext}
                status={slice.status}
                error={slice.error}
                teamId={teamId}
            />
        </WidgetCard>
    );
};
