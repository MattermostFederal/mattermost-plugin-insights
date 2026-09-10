// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/least_active_channels/channel_actions_menu/channel_actions_menu.tsx
// (commit 26617fcbdc).
//
// Adaptations:
//   - `MenuWrapper`/`Menu` (host-components) → a self-managed open/close
//     popover (same shape we use in `ScopeSelect`).
//   - `dispatch(leaveChannel(channelId))` /
//     `dispatch(joinChannel(...))` → direct REST calls against the host
//     server (`POST/DELETE /api/v4/channels/{id}/members`). The deprecated
//     code dispatched mattermost-redux thunks; the plugin bypasses redux
//     because we're not bundling mattermost-redux.
//   - `LeaveChannelModal` (host-component) → plugin-local
//     `LeaveChannelConfirmModal`.
//   - "Copy link" uses `navigator.clipboard.writeText()` against the
//     channel permalink. The deprecated code used the host's
//     `copyToClipboard` util which falls back to a textarea+execCommand
//     for older browsers; modern browsers all support the Clipboard API.

import React, {memo, useCallback, useEffect, useRef, useState} from 'react';
import {useIntl} from 'react-intl';

import {requestHeaders} from '../../client/csrf';
import type {TopInactiveChannel} from '../../types';
import {trackInsightsEvent} from '../../utils/telemetry';
import {LeaveChannelConfirmModal} from '../Modal/LeaveChannelConfirmModal';

interface Props {
    channel: TopInactiveChannel;
    teamName: string;
    currentUserId: string;
    isMember: boolean;
    onAfterLeave?: () => void;
}

async function postLeaveChannel(channelID: string, userID: string): Promise<void> {
    const resp = await fetch(`/api/v4/channels/${encodeURIComponent(channelID)}/members/${encodeURIComponent(userID)}`, {
        method: 'DELETE',
        credentials: 'include',
        headers: requestHeaders('DELETE'),
    });
    if (!resp.ok) {
        throw new Error(`leave channel failed: ${resp.status}`);
    }
}

const ChannelActionsMenuComponent: React.FC<Props> = ({channel, teamName, currentUserId, isMember, onAfterLeave}) => {
    const intl = useIntl();
    const [open, setOpen] = useState(false);
    const [confirmOpen, setConfirmOpen] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);

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

    const isPrivate = channel.type === 'P';
    const isDefaultChannel = channel.name === 'town-square';

    const doLeave = useCallback(async () => {
        await postLeaveChannel(channel.id, currentUserId);
        onAfterLeave?.();
    }, [channel.id, currentUserId, onAfterLeave]);

    const handleLeaveClick = useCallback((e: React.MouseEvent) => {
        e.stopPropagation();
        setOpen(false);
        trackInsightsEvent('leave_channel_action');
        if (isPrivate) {
            setConfirmOpen(true);
            return;
        }
        void doLeave();
    }, [doLeave, isPrivate]);

    const handleCopyLink = useCallback((e: React.MouseEvent) => {
        e.stopPropagation();
        setOpen(false);
        trackInsightsEvent('copy_channel_link_action');
        const url = `${window.location.origin}/${teamName}/channels/${channel.name}`;
        if (navigator.clipboard && navigator.clipboard.writeText) {
            void navigator.clipboard.writeText(url);
        }
    }, [channel.name, teamName]);

    const handleToggle = useCallback((e: React.MouseEvent) => {
        // Stop the event from reaching the row's own click handler (which
        // would navigate to the channel before the menu can open).
        e.stopPropagation();
        setOpen((s) => !s);
    }, []);

    return (
        <>
            <div
                ref={containerRef}
                className={`channel-action${open ? ' channel-action--open' : ''}`}
            >
                <button
                    type='button'
                    className='channel-action__toggle'
                    aria-label={intl.formatMessage({id: 'insights.leastActiveChannels.menuButtonAriaLabel', defaultMessage: 'Open manage channel menu'})}
                    aria-haspopup='menu'
                    aria-expanded={open}
                    onClick={handleToggle}
                >
                    <i className='icon icon-dots-vertical'/>
                </button>
                {open ? (
                    <div
                        className='channel-action__menu'
                        role='menu'
                        aria-label={intl.formatMessage({id: 'insights.leastActiveChannels.menuAriaLabel', defaultMessage: 'Manage channel menu'})}
                    >
                        {isMember && !isDefaultChannel ? (
                            <button
                                type='button'
                                role='menuitem'
                                className='channel-action__menu-item channel-action__menu-item--danger'
                                onClick={handleLeaveClick}
                            >
                                <span className='icon'>
                                    <i className='icon icon-logout-variant'/>
                                </span>
                                {intl.formatMessage({id: 'insights.leastActiveChannels.leaveChannel', defaultMessage: 'Leave channel'})}
                            </button>
                        ) : null}
                        <button
                            type='button'
                            role='menuitem'
                            className='channel-action__menu-item'
                            onClick={handleCopyLink}
                        >
                            <span className='icon'>
                                <i className='icon icon-link-variant'/>
                            </span>
                            {intl.formatMessage({id: 'insights.leastActiveChannels.copyLink', defaultMessage: 'Copy link'})}
                        </button>
                    </div>
                ) : null}
            </div>
            {confirmOpen ? (
                <LeaveChannelConfirmModal
                    show={true}
                    channelDisplayName={channel.display_name || channel.name}
                    onConfirm={doLeave}
                    onExited={() => setConfirmOpen(false)}
                />
            ) : null}
        </>
    );
};

export const ChannelActionsMenu = memo(ChannelActionsMenuComponent);
