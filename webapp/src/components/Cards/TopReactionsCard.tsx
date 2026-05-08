import React, {useEffect} from 'react';
import {FormattedMessage} from 'react-intl';
import {useDispatch, useSelector} from 'react-redux';

import {requestTopReactions} from '../../redux/actions';
import {selectTopReactions} from '../../redux/selectors';
import type {Scope, TimeRange} from '../../types';
import {preloadEmojis} from '../../utils/emoji';
import {WidgetCard} from '../Card/widgetHoc';
import {TopReactionsBarChart} from '../Charts/TopReactionsBarChart';
import {WidgetEmptyState} from '../EmptyState/WidgetEmptyState';
import {TopReactionsSkeleton} from '../Skeleton/CardSkeletons';

interface Props {
    scope: Scope;
    timeRange: TimeRange;

    // teamId is required for team scope and optional for user scope (acts as
    // an additional filter when present).
    teamId: string;

    onOpenDetails?: () => void;
}

const userScopeKey = (teamId: string) => teamId || 'global';

export const TopReactionsCard: React.FC<Props> = ({scope, timeRange, teamId, onOpenDetails}) => {
    const dispatch = useDispatch();
    const scopeKey = scope === 'team' ? teamId : userScopeKey(teamId);
    const slice = useSelector((state: Record<string, unknown>) => selectTopReactions(state, scope, scopeKey, timeRange));

    useEffect(() => {
        if (scope === 'team' && !teamId) {
            return;
        }

        // Cast: the redux-thunk middleware that ships with the Mattermost
        // webapp accepts function actions; the typings here intentionally
        // stay narrow so the action creators above remain self-contained.
        (dispatch as unknown as (a: unknown) => void)(requestTopReactions(scope, scopeKey, timeRange, scope === 'my' ? teamId : undefined));
    }, [dispatch, scope, scopeKey, timeRange, teamId]);

    // Mirrors the deprecated TopReactions dispatch:
    //   dispatch(loadCustomEmojisIfNeeded(reactions.map(r => r.emoji_name)))
    // Warms the browser cache for any custom emojis in the result so the
    // bar chart's <img>s render without flicker.
    useEffect(() => {
        if (slice.items.length) {
            preloadEmojis(slice.items.map((r) => r.emoji_name));
        }
    }, [slice.items]);

    let body: React.ReactNode;
    if (slice.status === 'loading') {
        body = <TopReactionsSkeleton/>;
    } else if (slice.status === 'error') {
        body = (
            <p className='insights-card__error'>{slice.error || (
                <FormattedMessage
                    id='insights.topReactions.failed'
                    defaultMessage='Failed to load reactions'
                />
            )}</p>
        );
    } else if (slice.items.length === 0) {
        body = <WidgetEmptyState icon='emoticon-outline'/>;
    } else {
        body = <TopReactionsBarChart reactions={slice.items.slice(0, 5)}/>;
    }

    return (
        <WidgetCard
            widgetType='topReactions'
            scope={scope}
            onOpenDetails={onOpenDetails}
        >
            {body}
        </WidgetCard>
    );
};
