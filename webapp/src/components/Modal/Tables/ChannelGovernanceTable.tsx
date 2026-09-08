// Container half of the channel-governance table: fetching, pagination, and
// navigation. The rendering lives in ChannelGovernanceList so it stays
// mountable without Redux in the CT suite.

import React, {memo, useCallback, useState} from 'react';
import {useSelector} from 'react-redux';

import {ChannelGovernanceList} from './ChannelGovernanceList';
import type {TableProps} from './types';

import {Client} from '../../../client/Client';
import {getCurrentTeamName} from '../../../redux/mmSelectors';
import type {ChannelActivity, ChannelGovernanceResponse, ChannelGovernanceSummary} from '../../../types';
import {navigateTo} from '../../../utils/navigation';
import {trackInsightsEvent} from '../../../utils/telemetry';
import {usePaginatedTable} from '../usePaginatedTable';

const ChannelGovernanceTableComponent: React.FC<TableProps> = ({timeRange, teamId}) => {
    const teamName = useSelector(getCurrentTeamName);
    const [summary, setSummary] = useState<ChannelGovernanceSummary | undefined>();

    // Team scope only — there is no per-user variant of the governance table,
    // and personal insights are disabled in 1.0 regardless.
    const fetcher = useCallback(
        (page: number, perPage: number) => Client.getChannelGovernance(teamId, timeRange, {page, perPage}),
        [teamId, timeRange],
    );

    const table = usePaginatedTable<ChannelActivity, ChannelGovernanceResponse>(
        fetcher,
        [teamId, timeRange],
        (resp) => setSummary(resp.summary),
    );

    const handleSelect = useCallback((channel: ChannelActivity) => {
        if (!teamName) {
            return;
        }
        trackInsightsEvent('open_channel_from_governance_table');
        navigateTo(`/${teamName}/channels/${channel.name}`);
    }, [teamName]);

    return (
        <ChannelGovernanceList
            items={table.items}
            summary={summary}
            loading={table.loading}
            error={table.error}
            onSelectChannel={teamName ? handleSelect : undefined}
        />
    );
};

export const ChannelGovernanceTable = memo(ChannelGovernanceTableComponent);
