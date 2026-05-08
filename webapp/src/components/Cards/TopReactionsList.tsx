import React from 'react';

import type {Status} from '../../redux/types';
import type {TopReaction} from '../../types';
import {WidgetEmptyState} from '../EmptyState/WidgetEmptyState';
import {TopReactionsSkeleton} from '../Skeleton/CardSkeletons';

interface Props {
    items: TopReaction[];
    hasNext: boolean;
    status: Status;
    error?: string;
}

// TopReactionsList is a presentational component: it renders the supplied
// items but does not fetch them. The container (TopReactionsCard) is
// responsible for dispatching the Redux thunk. Splitting the two keeps this
// piece trivially testable without a Redux Provider.
export const TopReactionsList: React.FC<Props> = ({items, status, error}) => {
    if (status === 'loading') {
        return <TopReactionsSkeleton/>;
    }
    if (status === 'error') {
        return <p className='insights-card__error'>{error || 'Failed to load reactions'}</p>;
    }
    if (items.length === 0) {
        return <WidgetEmptyState icon='emoticon-outline'/>;
    }
    return (
        <ol className='top-reactions'>
            {items.map((item, idx) => (
                <li
                    key={item.emoji_name}
                    className='top-reactions__item'
                >
                    <span className='top-reactions__rank'>{idx + 1}</span>
                    <span className='top-reactions__emoji'>{item.emoji_name}</span>
                    <span className='top-reactions__count'>{item.count}</span>
                </li>
            ))}
        </ol>
    );
};
