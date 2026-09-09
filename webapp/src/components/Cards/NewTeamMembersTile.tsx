// New Team Members as a stat tile rather than a card.
//
// It used to sit below the governance table as a full card, where six joiners
// took more vertical space than eleven channels — the minor feature
// out-weighing the major one. As a tile it joins the summary row instead.
//
// The count alone would delete the useful part, which is *who* joined, so the
// tile carries a face pile and opens the existing paginated modal on click.
// Names stay one click away rather than being thrown out.

import React, {memo, useEffect} from 'react';
import {FormattedMessage} from 'react-intl';
import {useDispatch, useSelector} from 'react-redux';

import {requestNewTeamMembers} from '../../redux/actions';
import {selectNewTeamMembers} from '../../redux/selectors';
import type {TimeRange} from '../../types';
import {HostAvatar} from '../Avatar/HostAvatar';

const FACE_PILE_LIMIT = 4;

interface Props {
    teamId: string;
    timeRange: TimeRange;
    onOpenDetails?: () => void;
}

const NewTeamMembersTileComponent: React.FC<Props> = ({teamId, timeRange, onOpenDetails}) => {
    const dispatch = useDispatch();
    const slice = useSelector((state: Record<string, unknown>) => selectNewTeamMembers(state, teamId, timeRange));

    useEffect(() => {
        if (!teamId) {
            return;
        }
        (dispatch as unknown as (a: unknown) => void)(requestNewTeamMembers(teamId, timeRange));
    }, [dispatch, teamId, timeRange]);

    const faces = slice.items.slice(0, FACE_PILE_LIMIT);
    const overflow = slice.totalCount - faces.length;

    const body = (
        <>
            <span className='governance-stat__value'>{slice.totalCount}</span>
            <span className='governance-stat__label'>
                <FormattedMessage
                    id='insights.governance.newMembers'
                    defaultMessage='Joined the team'
                />
            </span>
            {faces.length > 0 && (
                <span
                    className='governance-facepile'
                    data-testid='governance-facepile'
                >
                    {faces.map((m) => (
                        <HostAvatar
                            key={m.id}
                            userID={m.id}
                            lastPictureUpdate={m.last_picture_update}
                            size='sm'
                            className='governance-facepile__face'
                        />
                    ))}
                    {overflow > 0 && (
                        <span className='governance-facepile__more'>{`+${overflow}`}</span>
                    )}
                </span>
            )}
        </>
    );

    if (!onOpenDetails) {
        return <div className='governance-stat'>{body}</div>;
    }

    return (
        <button
            type='button'
            className='governance-stat governance-stat--button'
            data-testid='governance-new-members'
            onClick={onOpenDetails}
        >
            {body}
        </button>
    );
};

export const NewTeamMembersTile = memo(NewTeamMembersTileComponent);
