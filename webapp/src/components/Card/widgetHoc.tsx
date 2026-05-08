// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/widget_hoc/widget_hoc.tsx
// (commit 26617fcbdc).
//
// The deprecated HOC computed title/subtitle from `InsightsCardTitles`
// (a constant in `utils/constants.tsx`) and dispatched
// `openModal({modalId: ModalIdentifiers.INSIGHTS, dialogType: InsightsModal})`
// on click. Adaptations here:
//   - The InsightsModal is not opened via the host's `openModal` action;
//     this plugin keeps modal state in `InsightsPage` (see
//     `setModal`/`closeDetails` there) and the HOC just calls a passed
//     `onOpenDetails` callback.
//   - `InsightsCardTitles` is ported verbatim into `cardTitles` below
//     (with the trailing-`_widget` and trailing-`_modal` event suffixes
//     handled in `InsightsPage`/the modal tables, not here).

import React, {memo} from 'react';
import {FormattedMessage} from 'react-intl';

import {InsightCard} from './InsightCard';

import type {Scope} from '../../types';

export type InsightsWidgetType =
    | 'topReactions'
    | 'topChannels'
    | 'topThreads'
    | 'topDms'
    | 'topInactiveChannels'
    | 'newTeamMembers'
    | 'topBoards'
    | 'topPlaybooks';

interface CardTitleEntry {
    teamTitle: {id: string; defaultMessage: string};
    myTitle: {id: string; defaultMessage: string};
    teamSubTitle: {id: string; defaultMessage: string};
    mySubTitle: {id: string; defaultMessage: string};
}

// Verbatim port of the deprecated `InsightsCardTitles` constant from
// webapp/channels/src/utils/constants.tsx (commit 26617fcbdc^).
export const cardTitles: Record<InsightsWidgetType, CardTitleEntry> = {
    topChannels: {
        teamTitle: {id: 'insights.topChannels.title', defaultMessage: 'Top channels'},
        myTitle: {id: 'insights.topChannels.myTitle', defaultMessage: 'My top channels'},
        teamSubTitle: {id: 'insights.topChannels.subTitle', defaultMessage: 'Most active channels for the team'},
        mySubTitle: {id: 'insights.topChannels.mySubTitle', defaultMessage: "Most active channels that I'm a member of"},
    },
    topReactions: {
        teamTitle: {id: 'insights.topReactions.title', defaultMessage: 'Top reactions'},
        myTitle: {id: 'insights.topReactions.myTitle', defaultMessage: 'My top reactions'},
        teamSubTitle: {id: 'insights.topReactions.subTitle', defaultMessage: "The team's most-used reactions"},
        mySubTitle: {id: 'insights.topReactions.mySubTitle', defaultMessage: "Reactions I've used the most"},
    },
    topThreads: {
        teamTitle: {id: 'insights.topThreads.title', defaultMessage: 'Top threads'},
        myTitle: {id: 'insights.topThreads.myTitle', defaultMessage: 'My top threads'},
        teamSubTitle: {id: 'insights.topThreads.subTitle', defaultMessage: 'Most active threads for the team'},
        mySubTitle: {id: 'insights.topThreads.mySubTitle', defaultMessage: "Most active threads I've followed"},
    },
    topBoards: {
        teamTitle: {id: 'insights.topBoards.title', defaultMessage: 'Top boards'},
        myTitle: {id: 'insights.topBoards.myTitle', defaultMessage: 'My top boards'},
        teamSubTitle: {id: 'insights.topBoards.subTitle', defaultMessage: 'Most active boards for the team'},
        mySubTitle: {id: 'insights.topBoards.mySubTitle', defaultMessage: "Most active boards I've participated in"},
    },
    topInactiveChannels: {
        teamTitle: {id: 'insights.leastActiveChannels.title', defaultMessage: 'Least active channels'},
        myTitle: {id: 'insights.leastActiveChannels.myTitle', defaultMessage: 'My least active channels'},
        teamSubTitle: {id: 'insights.leastActiveChannels.subTitle', defaultMessage: 'Channels with the least posts'},
        mySubTitle: {id: 'insights.leastActiveChannels.mySubTitle', defaultMessage: 'My channels with the least posts'},
    },
    topPlaybooks: {
        teamTitle: {id: 'insights.topPlaybooks.title', defaultMessage: 'Top playbooks'},
        myTitle: {id: 'insights.topPlaybooks.myTitle', defaultMessage: 'My top playbooks'},
        teamSubTitle: {id: 'insights.topPlaybooks.subTitle', defaultMessage: 'Playbooks with the most runs'},
        mySubTitle: {id: 'insights.topPlaybooks.mySubTitle', defaultMessage: "Playbooks I've used with the most runs"},
    },
    topDms: {
        teamTitle: {id: 'insights.topDMs.title', defaultMessage: 'Top DMs'},
        myTitle: {id: 'insights.topDMs.myTitle', defaultMessage: 'My most active direct messages'},
        teamSubTitle: {id: 'insights.topDMs.subTitle', defaultMessage: ''},
        mySubTitle: {id: 'insights.topDMs.mySubTitle', defaultMessage: ''},
    },
    newTeamMembers: {
        teamTitle: {id: 'insights.newTeamMembers.title', defaultMessage: 'New team members'},
        myTitle: {id: 'insights.newTeamMembers.title', defaultMessage: 'New team members'},
        teamSubTitle: {id: 'insights.newMembers.subTitle', defaultMessage: 'People who recently joined the team'},
        mySubTitle: {id: 'insights.newMembers.subTitle', defaultMessage: 'People who recently joined the team'},
    },
};

export interface WidgetHocProps {
    widgetType: InsightsWidgetType;
    scope: Scope;
    onOpenDetails?: () => void;
    children: React.ReactNode;
}

const WidgetHocComponent: React.FC<WidgetHocProps> = ({widgetType, scope, onOpenDetails, children}) => {
    const entry = cardTitles[widgetType];
    const title = scope === 'team' ? entry.teamTitle : entry.myTitle;
    const subTitle = scope === 'team' ? entry.teamSubTitle : entry.mySubTitle;

    return (
        <InsightCard
            title={
                <FormattedMessage
                    id={title.id}
                    defaultMessage={title.defaultMessage}
                />
            }
            subtitle={subTitle.defaultMessage ? (
                <FormattedMessage
                    id={subTitle.id}
                    defaultMessage={subTitle.defaultMessage}
                />
            ) : undefined}
            onClick={onOpenDetails}
        >
            {children}
        </InsightCard>
    );
};

// WidgetCard wraps an insight card body with the centralized
// title/subtitle map and the click-to-open-details affordance. Mirrors the
// deprecated `widgetHoc` HOC's role.
export const WidgetCard = memo(WidgetHocComponent);
