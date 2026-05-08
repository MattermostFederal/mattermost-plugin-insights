// HostAvatar wraps the Mattermost host's `Avatar` and `imageURLForUser`
// when those are exposed at runtime via `window.Components` (see
// webapp/channels/src/plugins/export.ts in the host). The deprecated
// insights cards rendered:
//
//   <Avatar url={imageURLForUser(userID, lastPictureUpdate)} size='xs|xl'/>
//
// We replicate that exactly here when the host globals are present so the
// deprecated row layouts port verbatim. When the globals are missing
// (e.g. running outside the host or in CT tests) we fall back to a
// minimal `<img>` against the standard Mattermost user-image URL pattern.

import React, {memo} from 'react';

type Size = 'xxs' | 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'xxl';

const SIZE_PX: Record<Size, number> = {
    xxs: 16,
    xs: 20,
    sm: 24,
    md: 32,
    lg: 36,
    xl: 50,
    xxl: 72,
};

interface MMComponents {
    imageURLForUser?: (userID: string, lastPictureUpdate?: number) => string;
    Avatar?: React.ComponentType<{url?: string; username?: string; size?: Size; className?: string}>;
}

function host(): {Components?: MMComponents} {
    if (typeof window === 'undefined') {
        return {};
    }
    return window as unknown as {Components?: MMComponents};
}

interface Props {
    userID: string;
    lastPictureUpdate?: number;
    size?: Size;
    className?: string;
}

export function imageURLForUser(userID: string, lastPictureUpdate = 0): string {
    const helper = host().Components?.imageURLForUser;
    if (helper) {
        return helper(userID, lastPictureUpdate);
    }
    const cacheBust = lastPictureUpdate ? `?_=${lastPictureUpdate}` : '';
    return `/api/v4/users/${userID}/image${cacheBust}`;
}

const HostAvatarComponent: React.FC<Props> = ({userID, lastPictureUpdate, size = 'xs', className}) => {
    const HostComp = host().Components?.Avatar;
    const url = imageURLForUser(userID, lastPictureUpdate);
    if (HostComp) {
        return (
            <HostComp
                url={url}
                size={size}
                className={className}
            />
        );
    }
    const px = SIZE_PX[size];
    return (
        <img
            className={`Avatar Avatar___${size}${className ? ` ${className}` : ''}`}
            src={url}
            width={px}
            height={px}
            alt=''
        />
    );
};

export const HostAvatar = memo(HostAvatarComponent);
