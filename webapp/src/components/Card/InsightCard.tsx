import React from 'react';
import {useIntl} from 'react-intl';

interface Props {
    title: React.ReactNode;
    subtitle?: React.ReactNode;
    onClick?: () => void;
    ariaLabel?: string;
    children?: React.ReactNode;
}

export const InsightCard: React.FC<Props> = ({title, subtitle, onClick, ariaLabel, children}) => {
    const intl = useIntl();
    const clickable = typeof onClick === 'function';

    const handleHeaderClick = () => {
        if (onClick) {
            onClick();
        }
    };

    const handleButtonClick = (e: React.MouseEvent) => {
        e.stopPropagation();
        if (onClick) {
            onClick();
        }
    };

    const expandLabel = ariaLabel ?? intl.formatMessage(
        {id: 'insights.card.ariaLabel', defaultMessage: 'Open details for {title}'},
        {title: typeof title === 'string' ? title : ''},
    );

    return (
        <section className={`insights-card${clickable ? ' insights-card--clickable' : ''}`}>
            <header
                className='insights-card__header'
                onClick={clickable ? handleHeaderClick : undefined}
                role={clickable ? 'button' : undefined}
                tabIndex={clickable ? 0 : undefined}
                onKeyDown={clickable ? (e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        handleHeaderClick();
                    }
                } : undefined}
            >
                <div className='insights-card__title-block'>
                    <h2 className='insights-card__title'>{title}</h2>
                    {subtitle ? (
                        <div className='insights-card__subtitle'>{subtitle}</div>
                    ) : null}
                </div>
                {clickable ? (
                    <button
                        type='button'
                        className='insights-card__expand'
                        onClick={handleButtonClick}
                        aria-label={expandLabel}
                        aria-haspopup='dialog'
                    >
                        <i className='icon icon-chevron-right'/>
                    </button>
                ) : null}
            </header>
            <div className='insights-card__body'>
                {children}
            </div>
        </section>
    );
};
