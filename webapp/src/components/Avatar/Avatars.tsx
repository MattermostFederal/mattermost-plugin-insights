// Avatars (plural) — the "stack of avatars + +N overflow pill" used by
// the deprecated TopBoards / TopInactiveChannels / TopBoardsTable rows.
// Replaces the deprecated `<Avatars userIds={ids} size='xs'/>` host
// component (which we can't import). Visual structure matches the
// deprecated `.Avatars___xs` SCSS.

import React, {memo} from 'react';

import {HostAvatar} from './HostAvatar';

interface Props {
    userIDs: string[];
    size?: 'xs' | 'sm' | 'md';
    max?: number;
}

const AvatarsComponent: React.FC<Props> = ({userIDs, size = 'xs', max = 4}) => {
    const visible = userIDs.slice(0, max);
    const overflow = userIDs.length - visible.length;
    return (
        <div className={`Avatars Avatars___${size}`}>
            {visible.map((id) => (
                <HostAvatar
                    key={id}
                    userID={id}
                    size={size}
                />
            ))}
            {overflow > 0 ? (
                <span className='Avatars__overflow'>{`+${overflow}`}</span>
            ) : null}
        </div>
    );
};

export const Avatars = memo(AvatarsComponent);
