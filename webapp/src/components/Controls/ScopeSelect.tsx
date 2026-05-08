// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/insights_title/insights_title.tsx
// (commit 26617fcbdc).
//
// The deprecated source rendered the title as a `MenuWrapper` button with
// chevron-down and a small `Menu` of two `Menu.ItemAction`s for switching
// between My / Team Insights. Host-side `MenuWrapper` is not available
// here, so we manage open/close state locally and render the menu as a
// simple absolutely-positioned popover with the same icon + label set.
//
// License-aware behavior is ported faithfully:
//   - On a starter / free-trial license the Team Insights menu item is
//     marked with a lock indicator. Clicking it does NOT switch scope
//     (the deprecated `disabled={isStarterFree}` attr); it opens the
//     access modal CTA instead, mirroring the deprecated
//     `RestrictedIndicator` + `FeatureRestrictedModal` flow.
//   - Pre-trial vs post-trial vs end-user mode is selected via the
//     deprecated rules: admin + isFreeTrial=false → adminPreTrial,
//     admin + isFreeTrial=true → adminPostTrial, otherwise endUser.

import React, {memo, useCallback, useEffect, useRef, useState} from 'react';
import {FormattedMessage, useIntl} from 'react-intl';
import {useSelector} from 'react-redux';

import {useLicenseChecks} from '../../hooks/useLicenseChecks';
import type {Scope} from '../../types';
import {AccessModal} from '../AccessModal/AccessModal';
import type {AccessModalMode} from '../AccessModal/AccessModal';

interface Props {
    value: Scope;
    onChange: (scope: Scope) => void;
}

interface RolesUserState {
    entities?: {
        users?: {
            currentUserId?: string;
            profiles?: Record<string, {roles?: string}>;
        };
    };
}

function isCurrentUserAdmin(state: unknown): boolean {
    const root = state as RolesUserState;
    const id = root.entities?.users?.currentUserId ?? '';
    const profile = root.entities?.users?.profiles?.[id];
    const roles = profile?.roles ?? '';
    return roles.split(' ').includes('system_admin');
}

const ScopeSelectComponent: React.FC<Props> = ({value, onChange}) => {
    const intl = useIntl();
    const [open, setOpen] = useState(false);
    const [accessModalOpen, setAccessModalOpen] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);

    const {isStarterFree, isFreeTrial} = useLicenseChecks();
    const isAdmin = useSelector(isCurrentUserAdmin);
    const teamInsightsRestricted = isStarterFree;

    useEffect(() => {
        if (!open) {
            return undefined;
        }
        const handleDocClick = (e: MouseEvent) => {
            if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
                setOpen(false);
            }
        };
        document.addEventListener('mousedown', handleDocClick);
        return () => document.removeEventListener('mousedown', handleDocClick);
    }, [open]);

    const select = useCallback((next: Scope) => {
        if (next === 'team' && teamInsightsRestricted) {
            setOpen(false);
            setAccessModalOpen(true);
            return;
        }
        onChange(next);
        setOpen(false);
    }, [onChange, teamInsightsRestricted]);

    let modalMode: AccessModalMode;
    if (isAdmin && !isFreeTrial) {
        modalMode = 'adminPreTrial';
    } else if (isAdmin) {
        modalMode = 'adminPostTrial';
    } else {
        modalMode = 'endUser';
    }

    return (
        <div
            ref={containerRef}
            className={`insights-scope${open ? ' insights-scope--open' : ''}`}
        >
            <button
                type='button'
                className='insights-scope__title'
                aria-haspopup='menu'
                aria-expanded={open}
                onClick={() => setOpen((s) => !s)}
            >
                {value === 'team' ? (
                    <FormattedMessage
                        id='insights.teamHeading'
                        defaultMessage='Team Insights'
                    />
                ) : (
                    <FormattedMessage
                        id='insights.myHeading'
                        defaultMessage='My Insights'
                    />
                )}
                <span className='icon'>
                    <i className='icon icon-chevron-down'/>
                </span>
            </button>
            {open ? (
                <div
                    className='insights-scope__menu'
                    role='menu'
                    aria-label={intl.formatMessage({id: 'insights.filter.ariaLabel', defaultMessage: 'Insights filter menu'})}
                >
                    <button
                        type='button'
                        role='menuitem'
                        className={`insights-scope__menu-item${value === 'my' ? ' is-selected' : ''}`}
                        onClick={() => select('my')}
                    >
                        <span className='icon'>
                            <i className='icon icon-account-outline'/>
                        </span>
                        <FormattedMessage
                            id='insights.filter.myInsights'
                            defaultMessage='My Insights'
                        />
                    </button>
                    <button
                        type='button'
                        role='menuitem'
                        className={`insights-scope__menu-item${value === 'team' ? ' is-selected' : ''}${teamInsightsRestricted ? ' is-restricted' : ''}`}
                        aria-disabled={teamInsightsRestricted ? 'true' : undefined}
                        onClick={() => select('team')}
                    >
                        <span className='icon'>
                            <i className='icon icon-account-multiple-outline'/>
                        </span>
                        <FormattedMessage
                            id='insights.filter.teamInsights'
                            defaultMessage='Team Insights'
                        />
                        {teamInsightsRestricted ? (
                            <span
                                className='insights-scope__restricted-icon'
                                aria-label={intl.formatMessage({id: 'insights.accessModal.cloudFreeTrial', defaultMessage: 'During your trial you are able to view Team Insights.'})}
                            >
                                <i className='icon icon-lock-outline'/>
                            </span>
                        ) : null}
                    </button>
                </div>
            ) : null}
            {accessModalOpen ? (
                <AccessModal
                    show={true}
                    mode={modalMode}
                    onExited={() => setAccessModalOpen(false)}
                />
            ) : null}
        </div>
    );
};

export const ScopeSelect = memo(ScopeSelectComponent);
