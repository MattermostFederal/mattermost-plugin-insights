// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/insights_modal/insights_modal.tsx
// (commit 26617fcbdc).
//
// Adaptations:
//   - The deprecated source ran inside the host webapp, so it imported the
//     six per-insight tables directly. We do the same — each table is a
//     plugin-local component under `Modal/Tables/`. The widget-type switch
//     mirrors the deprecated InsightsWidgetTypes union exactly.
//   - The deprecated `setShowModal` callback exists for legacy upstream
//     state; here `onExited` is the close handler and we don't need a
//     separate setShowModal prop.

import React, {memo, useCallback, useState} from 'react';
import {Modal as BootstrapModal} from 'react-bootstrap';

// react-bootstrap 2.x's component types use the legacy `BsPrefixRefForwardingComponent`
// generic which React 18.x types reject. Cast to a permissive shape so the
// JSX usage type-checks; runtime behavior is unchanged (the component is
// resolved from `window.ReactBootstrap` via webpack externals).
const Modal = BootstrapModal as unknown as React.ComponentType<Record<string, unknown>> & {
    Header: React.ComponentType<Record<string, unknown>>;
    Title: React.ComponentType<Record<string, unknown>>;
    Body: React.ComponentType<Record<string, unknown>>;
};

import {LeastActiveChannelsTable} from './Tables/LeastActiveChannelsTable';
import {NewMembersTable} from './Tables/NewMembersTable';
import {TopBoardsTable} from './Tables/TopBoardsTable';
import {TopChannelsTable} from './Tables/TopChannelsTable';
import {TopDMsTable} from './Tables/TopDMsTable';
import {TopPlaybooksTable} from './Tables/TopPlaybooksTable';
import {TopReactionsTable} from './Tables/TopReactionsTable';
import {TopThreadsTable} from './Tables/TopThreadsTable';

import type {Scope, TimeRange} from '../../types';
import {TimeRangeSelect} from '../Controls/TimeRangeSelect';

export type InsightsWidgetType =
    | 'topReactions'
    | 'topChannels'
    | 'topThreads'
    | 'topDms'
    | 'topInactiveChannels'
    | 'newTeamMembers'
    | 'topPlaybooks'
    | 'topBoards';

interface Props {
    show: boolean;
    onExited: () => void;
    widgetType: InsightsWidgetType;
    title: string;
    subtitle: string;
    scope: Scope;
    timeRange: TimeRange;
    teamId: string;
}

const InsightsModalComponent: React.FC<Props> = (props) => {
    const [show, setShow] = useState(props.show);
    const [timeRange, setTimeRange] = useState<TimeRange>(props.timeRange);

    const handleHide = useCallback(() => {
        setShow(false);
    }, []);

    const handleTimeRangeChange = useCallback((next: TimeRange) => {
        setTimeRange(next);
    }, []);

    const renderBody = () => {
        const common = {scope: props.scope, timeRange, teamId: props.teamId};
        switch (props.widgetType) {
            case 'topReactions':
                return <TopReactionsTable {...common}/>;
            case 'topChannels':
                return <TopChannelsTable {...common}/>;
            case 'topThreads':
                return <TopThreadsTable {...common}/>;
            case 'topDms':
                return <TopDMsTable {...common}/>;
            case 'topInactiveChannels':
                return <LeastActiveChannelsTable {...common}/>;
            case 'newTeamMembers':
                return <NewMembersTable {...common}/>;
            case 'topPlaybooks':
                return <TopPlaybooksTable {...common}/>;
            case 'topBoards':
                return <TopBoardsTable {...common}/>;
            default:
                return null;
        }
    };

    return (
        <Modal
            dialogClassName='a11y__modal insights-modal'
            show={show}
            onHide={handleHide}
            onExited={props.onExited}
            aria-labelledby='insightsModalLabel'
            id='insightsModal'
        >
            <Modal.Header closeButton={true}>
                <div className='title-section'>
                    <Modal.Title
                        as='h1'
                        id='insightsModalTitle'
                    >
                        {props.title}
                    </Modal.Title>
                    <div className='subtitle'>{props.subtitle}</div>
                </div>
                <TimeRangeSelect
                    value={timeRange}
                    onChange={handleTimeRangeChange}
                />
            </Modal.Header>
            <Modal.Body className='overflow--visible'>
                {renderBody()}
            </Modal.Body>
        </Modal>
    );
};

export const InsightsModal = memo(InsightsModalComponent);
