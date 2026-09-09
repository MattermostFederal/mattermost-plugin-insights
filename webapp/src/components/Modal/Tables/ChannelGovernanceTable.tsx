// Container half of the channel-governance table: fetching, pagination, and
// navigation. The rendering lives in ChannelGovernanceList so it stays
// mountable without Redux in the CT suite.

import React, {memo, useCallback, useState} from 'react';
import {useSelector} from 'react-redux';

import {ChannelGovernanceList} from './ChannelGovernanceList';
import type {SortColumn} from './ChannelGovernanceList';
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
    const [sort, setSort] = useState<SortColumn>('posts');
    const [ascending, setAscending] = useState(false);

    // Team scope only — there is no per-user variant of the governance table,
    // and personal insights are disabled in 1.0 regardless.
    const fetcher = useCallback(
        (page: number, perPage: number) => Client.getChannelGovernance(teamId, timeRange, {
            page,
            perPage,
            sort,
            direction: ascending ? 'asc' : 'desc',
        }),
        [teamId, timeRange, sort, ascending],
    );

    const table = usePaginatedTable<ChannelActivity, ChannelGovernanceResponse>(
        fetcher,
        [teamId, timeRange, sort, ascending],
        (resp) => setSummary(resp.summary),
    );

    // Clicking the active column flips direction; clicking a new one starts
    // descending, since "most posts" is the usual first question.
    const handleSort = useCallback((column: SortColumn) => {
        if (column === sort) {
            setAscending((prev) => !prev);
            return;
        }
        setSort(column);
        setAscending(false);
    }, [sort]);

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
            sort={sort}
            ascending={ascending}
            onSort={handleSort}
        />
    );
};

export const ChannelGovernanceTable = memo(ChannelGovernanceTableComponent);
