import React from 'react';
import {FormattedMessage} from 'react-intl';

interface Props {
    icon: string;
    message?: React.ReactNode;
}

export const WidgetEmptyState: React.FC<Props> = ({icon, message}) => (
    <div className='insights-empty-state'>
        <div className='insights-empty-state__icon'>
            <i className={`icon icon-${icon}`}/>
        </div>
        <div className='insights-empty-state__text'>
            {message ?? (
                <FormattedMessage
                    id='insights.empty.default'
                    defaultMessage='Not enough data yet for this insight'
                />
            )}
        </div>
    </div>
);
