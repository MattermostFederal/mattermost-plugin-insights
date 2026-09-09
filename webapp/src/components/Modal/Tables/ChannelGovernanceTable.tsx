// Container half of the channel-governance table: fetching, pagination, and
// navigation. The rendering lives in ChannelGovernanceList so it stays
// mountable without Redux in the CT suite.

import React, {memo, useCallback, useEffect, useState} from 'react';
import {useSelector} from 'react-redux';

import {ChannelGovernanceList} from './ChannelGovernanceList';
import type {GovernanceFilter} from './ChannelGovernanceList';
import {nextSortState} from './sorting';
import type {SortColumn} from './sorting';
import type {TableProps} from './types';

import {Client} from '../../../client/Client';
import {getCurrentTeamName} from '../../../redux/mmSelectors';
import type {ChannelActivity, ChannelGovernanceResponse, ChannelGovernanceSummary} from '../../../types';
import {navigateTo} from '../../../utils/navigation';
import {trackInsightsEvent} from '../../../utils/telemetry';
import {AUDIT_PER_PAGE, usePaginatedTable} from '../usePaginatedTable';

// R5's debounce, landing where it actually matters: the search box, which is
// the only control a user can hammer.
function useDebounced<T>(value: T, delayMs: number): T {
    const [settled, setSettled] = useState(value);
    useEffect(() => {
        const id = setTimeout(() => setSettled(value), delayMs);
        return () => clearTimeout(id);
    }, [value, delayMs]);
    return settled;
}

interface GovernanceTableProps extends TableProps {
    trailingTile?: React.ReactNode;
}

const ChannelGovernanceTableComponent: React.FC<GovernanceTableProps> = ({timeRange, teamId, trailingTile}) => {
    const teamName = useSelector(getCurrentTeamName);
    const [summary, setSummary] = useState<ChannelGovernanceSummary | undefined>();
    const [sort, setSort] = useState<SortColumn>('posts');
    const [ascending, setAscending] = useState(false);
    const [search, setSearch] = useState('');
    const [filter, setFilter] = useState<GovernanceFilter>('');
    const [generatedAt, setGeneratedAt] = useState(0);

    // Typing re-queries on every keystroke otherwise. The request is cheap
    // (it filters an in-memory slice) but the re-render churn is not.
    const debouncedSearch = useDebounced(search, 250);

    // Team scope only — there is no per-user variant of the governance table,
    // and personal insights are disabled in 1.0 regardless.
    const fetcher = useCallback(
        (page: number, perPage: number) => Client.getChannelGovernance(teamId, timeRange, {
            page,
            perPage,
            sort,
            direction: ascending ? 'asc' : 'desc',
            search: debouncedSearch,
            filter,
        }),
        [teamId, timeRange, sort, ascending, debouncedSearch, filter],
    );

    const table = usePaginatedTable<ChannelActivity, ChannelGovernanceResponse>(
        fetcher,
        [teamId, timeRange, sort, ascending, debouncedSearch, filter],
        (resp) => {
            setSummary(resp.summary);
            setGeneratedAt(resp.generated_at);
        },
        AUDIT_PER_PAGE,
    );

    const handleSort = useCallback((column: SortColumn) => {
        const next = nextSortState({sort, ascending}, column);
        setSort(next.sort);
        setAscending(next.ascending);
    }, [sort, ascending]);

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
            search={search}
            onSearch={setSearch}
            filter={filter}
            onFilter={setFilter}
            generatedAt={generatedAt}
            trailingTile={trailingTile}
            timeRange={timeRange}
        />
    );
};

export const ChannelGovernanceTable = memo(ChannelGovernanceTableComponent);
