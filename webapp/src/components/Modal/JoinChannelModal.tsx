// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/join_channel_modal/join_channel_modal.tsx
// (commit 26617fcbdc^).
//
// Adaptations:
//   - `dispatch(joinChannel(userID, teamID, channelID, channelName))` →
//     direct REST call against the host server
//     (`POST /api/v4/channels/{channelID}/members`). The deprecated thunk
//     also dispatched several Redux events to keep the host's channel
//     store in sync; we accept that the host's WebSocket layer will
//     repopulate state shortly after the join succeeds.
//   - `dispatch(selectPost(thread.post))` → caller-supplied
//     `onJoined(thread)` callback. The TopThreads list passes a callback
//     that opens the RHS via the existing `openThreadRHS` helper.
//   - `<SaveButton>` → simple `<button>` with a busy-spinner emoji
//     placeholder; the SCSS matches the deprecated `.save-button` /
//     `.join-channel-cancel` block.

import React, {memo, useCallback, useState} from 'react';
import {Modal as BootstrapModal} from 'react-bootstrap';
import {FormattedMessage} from 'react-intl';

import type {TopThread} from '../../types';

const Modal = BootstrapModal as unknown as React.ComponentType<Record<string, unknown>> & {
    Header: React.ComponentType<Record<string, unknown>>;
    Title: React.ComponentType<Record<string, unknown>>;
    Body: React.ComponentType<Record<string, unknown>>;
};

interface Props {
    show: boolean;
    onExited: () => void;
    thread: TopThread;
    currentUserId: string;
    onJoined: (thread: TopThread) => void;
}

interface JoinError {
    message: string;
}

async function joinChannel(channelID: string, userID: string): Promise<JoinError | null> {
    try {
        const resp = await fetch(`/api/v4/channels/${encodeURIComponent(channelID)}/members`, {
            method: 'POST',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            body: JSON.stringify({user_id: userID}),
        });
        if (!resp.ok) {
            return {message: `join failed: ${resp.status} ${resp.statusText}`};
        }
        return null;
    } catch (err) {
        return {message: err instanceof Error ? err.message : String(err)};
    }
}

const JoinChannelModalComponent: React.FC<Props> = ({show, onExited, thread, currentUserId, onJoined}) => {
    const [open, setOpen] = useState(show);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string>('');

    const doHide = useCallback(() => setOpen(false), []);

    const handleJoin = useCallback(async () => {
        setSaving(true);
        setError('');
        const err = await joinChannel(thread.channel_id, currentUserId);
        setSaving(false);
        if (err) {
            setError(err.message);
            return;
        }
        onJoined(thread);
        setOpen(false);
    }, [thread, currentUserId, onJoined]);

    return (
        <Modal
            dialogClassName='a11y__modal insights-modal join-channel-modal'
            show={open}
            onHide={doHide}
            onExited={onExited}
            aria-labelledby='joinChannelModalLabel'
            id='joinChannelModal'
        >
            <Modal.Header closeButton={true}>
                <div className='title-section'>
                    <Modal.Title
                        as='h1'
                        id='joinChannelModalLabel'
                    >
                        <FormattedMessage
                            id='joinChannel.title'
                            defaultMessage='Join channel?'
                        />
                    </Modal.Title>
                </div>
            </Modal.Header>
            <Modal.Body className='overflow--visible'>
                <FormattedMessage
                    id='joinChannel.desciption'
                    defaultMessage="You'll need to join the {channel} channel to see this thread. Do you want to join {channel} now?"
                    values={{
                        channel: <strong>{thread.channel_display_name || thread.channel_name}</strong>,
                    }}
                />
                {error ? (
                    <p className='insights-card__error'>{error}</p>
                ) : null}
                <div className='button-footer'>
                    <button
                        type='button'
                        className='btn join-channel-cancel'
                        onClick={(e) => {
                            e.preventDefault();
                            doHide();
                        }}
                    >
                        <FormattedMessage
                            id='joinChannel.cancelButton'
                            defaultMessage='Cancel'
                        />
                    </button>
                    <button
                        type='button'
                        className='btn save-button'
                        disabled={saving}
                        onClick={(e) => {
                            e.preventDefault();
                            void handleJoin();
                        }}
                    >
                        {saving ? (
                            <FormattedMessage
                                id='joinChannel.joiningButton'
                                defaultMessage='Joining...'
                            />
                        ) : (
                            <FormattedMessage
                                id='joinChannel.JoinButton'
                                defaultMessage='Join'
                            />
                        )}
                    </button>
                </div>
            </Modal.Body>
        </Modal>
    );
};

export const JoinChannelModal = memo(JoinChannelModalComponent);
