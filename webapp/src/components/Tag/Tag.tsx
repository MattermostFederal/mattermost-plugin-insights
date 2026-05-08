// Slim port of mattermost/webapp `components/widgets/tag/tag.tsx`. The
// deprecated insights cards used the host's `<Tag text={channel.name}/>`
// (Channels' display-name pill). We can't import the host's
// styled-components version directly, so reproduce its structure as a
// simple inline-flex pill.

import React, {memo} from 'react';

interface Props {
    text: React.ReactNode;
    icon?: React.ReactNode;
    className?: string;
    onClick?: React.MouseEventHandler;
}

const TagComponent: React.FC<Props> = ({text, icon, className, onClick}) => (
    <span
        className={`Tag${onClick ? ' Tag--clickable' : ''}${className ? ` ${className}` : ''}`}
        role={onClick ? 'button' : undefined}
        tabIndex={onClick ? 0 : undefined}
        onClick={onClick}
    >
        {icon ? <span className='Tag__icon'>{icon}</span> : null}
        <span className='Tag__text'>{text}</span>
    </span>
);

export const Tag = memo(TagComponent);
