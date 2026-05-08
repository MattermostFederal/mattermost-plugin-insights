// Ported from mattermost/mattermost
// webapp/channels/src/components/feature_restricted_modal/feature_restricted_modal.tsx
// (commit 26617fcbdc) with the slim shape the deprecated
// `insights_title.tsx` actually invoked: a CTA that explains why team
// insights are restricted on a starter / free-trial license, with
// admin-vs-end-user message variants.
//
// Adaptations:
//   - Trial-start RPC and self-hosted-products selector are not available
//     to the plugin; we render the same CTA strings the deprecated
//     `accessModal` used and link out to the same docs / sales pages.
//   - The four admin-pre-trial / admin-post-trial / end-user variants are
//     selected by `mode`. Caller determines mode from `isFreeTrial` plus
//     the user's roles.

import React, {memo, useState} from 'react';
import {Modal as BootstrapModal} from 'react-bootstrap';
import {FormattedMessage} from 'react-intl';

const Modal = BootstrapModal as unknown as React.ComponentType<Record<string, unknown>> & {
    Header: React.ComponentType<Record<string, unknown>>;
    Title: React.ComponentType<Record<string, unknown>>;
    Body: React.ComponentType<Record<string, unknown>>;
    Footer: React.ComponentType<Record<string, unknown>>;
};

export type AccessModalMode = 'adminPreTrial' | 'adminPostTrial' | 'endUser';

interface Props {
    show: boolean;
    onExited: () => void;
    mode: AccessModalMode;
}

const TRIAL_LENGTH_DAYS = 30;

const TitleByMode = {
    adminPreTrial: (
        <FormattedMessage
            id='insights.accessModal.titleAdminPreTrial'
            defaultMessage='Try team insights with a free trial'
        />
    ),
    adminPostTrial: (
        <FormattedMessage
            id='insights.accessModal.titleAdminPostTrial'
            defaultMessage='Upgrade to access team insights'
        />
    ),
    endUser: (
        <FormattedMessage
            id='insights.accessModal.titleEndUser'
            defaultMessage='Team insights are available in paid plans'
        />
    ),
} as const;

const TeamInsightsLink = (chunks: React.ReactNode) => (
    <a
        href='https://docs.mattermost.com/welcome/insights.html#team-insights'
        target='_blank'
        rel='noopener noreferrer'
    >{chunks}</a>
);

const InsightsDocLink = (chunks: React.ReactNode) => (
    <a
        href='https://docs.mattermost.com/welcome/insights.html'
        target='_blank'
        rel='noopener noreferrer'
    >{chunks}</a>
);

const ContactSalesLink = (chunks: React.ReactNode) => (
    <a
        href='https://mattermost.com/contact-sales/'
        target='_blank'
        rel='noopener noreferrer'
    >{chunks}</a>
);

const AccessModalComponent: React.FC<Props> = ({show, onExited, mode}) => {
    const [open, setOpen] = useState(show);

    const handleHide = () => setOpen(false);

    let body: React.ReactNode;
    if (mode === 'adminPreTrial') {
        body = (
            <FormattedMessage
                id='insights.accessModal.messageAdminPreTrial'
                defaultMessage='Use <teamInsights>Team Insights</teamInsights> with one of our paid plans. Get the full experience of Enterprise when you start a free, {trialLength} day trial.'
                values={{
                    trialLength: TRIAL_LENGTH_DAYS,
                    teamInsights: TeamInsightsLink,
                }}
            />
        );
    } else if (mode === 'adminPostTrial') {
        body = (
            <FormattedMessage
                id='insights.accessModal.messageAdminPostTrial'
                defaultMessage='To access your complete <insightsDoc>Insights</insightsDoc> dashboard, including <teamInsights>Team Insights</teamInsights>, please upgrade your plan to Professional or Enterprise. For questions on upgrading your plan, please <contactSales>contact our Sales team</contactSales> for support.'
                values={{
                    insightsDoc: InsightsDocLink,
                    teamInsights: TeamInsightsLink,
                    contactSales: ContactSalesLink,
                }}
            />
        );
    } else {
        body = (
            <FormattedMessage
                id='insights.accessModal.messageEndUser'
                defaultMessage='To access your complete <insightsDoc>Insights</insightsDoc> dashboard, including <teamInsights>Team Insights</teamInsights>, please notify your Admin to upgrade your plan to Professional or Enterprise. For questions on upgrading your plan, please <contactSales>contact our Sales team</contactSales> for support.'
                values={{
                    insightsDoc: InsightsDocLink,
                    teamInsights: TeamInsightsLink,
                    contactSales: ContactSalesLink,
                }}
            />
        );
    }

    return (
        <Modal
            dialogClassName='a11y__modal insights-access-modal'
            show={open}
            onHide={handleHide}
            onExited={onExited}
        >
            <Modal.Header closeButton={true}>
                <Modal.Title as='h2'>
                    {TitleByMode[mode]}
                </Modal.Title>
            </Modal.Header>
            <Modal.Body>
                <p>{body}</p>
            </Modal.Body>
        </Modal>
    );
};

export const AccessModal = memo(AccessModalComponent);
