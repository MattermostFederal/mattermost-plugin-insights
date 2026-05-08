// Slim port of mattermost/webapp `components/leave_channel_modal` — used
// for the private-channel confirmation step the deprecated
// `channel_actions_menu` opened.

import React, {memo, useState} from 'react';
import {Modal as BootstrapModal} from 'react-bootstrap';
import {FormattedMessage} from 'react-intl';

const Modal = BootstrapModal as unknown as React.ComponentType<Record<string, unknown>> & {
    Header: React.ComponentType<Record<string, unknown>>;
    Title: React.ComponentType<Record<string, unknown>>;
    Body: React.ComponentType<Record<string, unknown>>;
    Footer: React.ComponentType<Record<string, unknown>>;
};

interface Props {
    show: boolean;
    onExited: () => void;
    channelDisplayName: string;
    onConfirm: () => Promise<void> | void;
}

const LeaveChannelConfirmModalComponent: React.FC<Props> = ({show, onExited, channelDisplayName, onConfirm}) => {
    const [open, setOpen] = useState(show);
    const [busy, setBusy] = useState(false);

    const handleHide = () => setOpen(false);

    const handleConfirm = async () => {
        setBusy(true);
        try {
            await onConfirm();
        } finally {
            setBusy(false);
            setOpen(false);
        }
    };

    return (
        <Modal
            dialogClassName='a11y__modal'
            show={open}
            onHide={handleHide}
            onExited={onExited}
        >
            <Modal.Header closeButton={true}>
                <Modal.Title as='h2'>
                    <FormattedMessage
                        id='leave_private_channel_modal.title'
                        defaultMessage='Leave private channel {channel}'
                        values={{channel: channelDisplayName}}
                    />
                </Modal.Title>
            </Modal.Header>
            <Modal.Body>
                <FormattedMessage
                    id='leave_private_channel_modal.message'
                    defaultMessage='Are you sure you wish to leave the private channel {channel}? You must be re-invited to rejoin.'
                    values={{channel: <strong>{channelDisplayName}</strong>}}
                />
            </Modal.Body>
            <Modal.Footer>
                <button
                    type='button'
                    className='btn btn-tertiary'
                    onClick={handleHide}
                >
                    <FormattedMessage
                        id='leave_private_channel_modal.cancel'
                        defaultMessage='Cancel'
                    />
                </button>
                <button
                    type='button'
                    className='btn btn-danger'
                    onClick={handleConfirm}
                    disabled={busy}
                >
                    <FormattedMessage
                        id='leave_private_channel_modal.leave'
                        defaultMessage='Yes, leave channel'
                    />
                </button>
            </Modal.Footer>
        </Modal>
    );
};

export const LeaveChannelConfirmModal = memo(LeaveChannelConfirmModalComponent);
