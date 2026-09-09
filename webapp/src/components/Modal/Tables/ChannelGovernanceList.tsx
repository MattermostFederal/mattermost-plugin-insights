// Presentational half of the channel-governance table. No Redux and no
// fetching, so the Playwright CT suite can mount it directly with fixtures.
// The Redux-bound container is ChannelGovernanceTable.tsx.
//
// Column set is the 1.0 agreed one: channel, type, posts, active users,
// members, last post, created, purpose. Tags and sidebar categories were
// requested but do not exist on the Mattermost channel model — see
// webapp/src/types.ts (ChannelActivity).

import React, {memo, useMemo} from 'react';
import {FormattedMessage} from 'react-intl';

import type {SortColumn} from './sorting';

import type {ChannelActivity, ChannelGovernanceSummary, TimeRange} from '../../../types';

export type {SortColumn};

export interface Props {
    items: ChannelActivity[];
    summary?: ChannelGovernanceSummary;
    loading?: boolean;
    error?: string;
    onSelectChannel?: (channel: ChannelActivity) => void;

    // Sorting is server-side because it has to order the whole team's
    // channels, not just the page on screen. It is cheap there: the rows are
    // already in memory in the daily snapshot, so no query runs.
    sort?: SortColumn;
    ascending?: boolean;
    onSort?: (column: SortColumn) => void;

    // Search and quick filters. At a few hundred channels, paging is not a
    // way to find anything — these are how the table is actually used.
    search?: string;
    onSearch?: (value: string) => void;
    filter?: GovernanceFilter;
    onFilter?: (value: GovernanceFilter) => void;

    // When the snapshot was built, unix millis. 0 means "just now".
    generatedAt?: number;

    // An extra tile appended to the summary row. The page passes New Team
    // Members here so it joins the stat row instead of floating below the
    // table; this component stays unaware of what is in it.
    trailingTile?: React.ReactNode;

    timeRange?: TimeRange;
}

export type GovernanceFilter = '' | 'unlabelled' | 'inactive' | 'private' | 'public';

function noPostsLabel(timeRange?: TimeRange): string {
    switch (timeRange) {
    case '1_day':
        return 'No posts since yesterday';
    case '7_day':
        return 'No posts in the last 7 days';
    case '28_day':
        return 'No posts in the last 28 days';
    default:
        return 'No posts in selected window';
    }
}

function formatDate(unixMillis: number): string {
    if (!unixMillis) {
        return '';
    }
    return new Date(unixMillis).toLocaleDateString([], {year: 'numeric', month: 'short', day: '2-digit'});
}

// A channel that has never been posted in carries last_post_at = 0, which
// would otherwise format as 1 Jan 1970.
function formatLastPost(unixMillis: number): React.ReactNode {
    if (!unixMillis) {
        return (
            <span className='governance-muted'>
                <FormattedMessage
                    id='insights.governance.never'
                    defaultMessage='Never'
                />
            </span>
        );
    }
    return formatDate(unixMillis);
}

const percent = (n: number, total: number): number => (total ? Math.round((n / total) * 100) : 0);

// The tiles state the work remaining rather than the work done: "6 need a
// purpose" is a queue, "5 are labelled" is trivia.
const CoverageSummary: React.FC<{summary: ChannelGovernanceSummary; trailingTile?: React.ReactNode; timeRange?: TimeRange}> = ({summary, trailingTile, timeRange}) => (
    <div
        className={`governance-summary${trailingTile ? ' governance-summary--four' : ''}`}
        data-testid='governance-summary'
    >
        <div className='governance-stat'>
            <span className='governance-stat__value'>{summary.total_channels}</span>
            <span className='governance-stat__label'>
                <FormattedMessage
                    id='insights.governance.totalChannels'
                    defaultMessage='Channels in team'
                />
            </span>
        </div>
        <div className='governance-stat'>
            <span className='governance-stat__value'>{summary.total_channels - summary.active_channels}</span>
            <span className='governance-stat__label'>
                {noPostsLabel(timeRange)}
            </span>
        </div>
        <div className='governance-stat'>
            <span className='governance-stat__value'>{summary.total_channels - summary.with_purpose}</span>
            <span className='governance-stat__label'>
                <FormattedMessage
                    id='insights.governance.missingPurpose'
                    defaultMessage='Missing a purpose'
                />
            </span>
            <span className='governance-stat__sub'>
                {`${percent(summary.with_purpose, summary.total_channels)}% of ${summary.total_channels} labelled`}
            </span>
        </div>
        {trailingTile}
    </div>
);

interface HeaderProps {
    column: SortColumn;
    label: React.ReactNode;
    numeric?: boolean;
    sort?: SortColumn;
    ascending?: boolean;
    onSort?: (column: SortColumn) => void;
}

const SortableHeader: React.FC<HeaderProps> = ({column, label, numeric, sort, ascending, onSort}) => {
    const active = sort === column;
    if (!onSort) {
        return <th className={numeric ? 'governance-cell--num' : undefined}>{label}</th>;
    }

    let ariaSort: 'ascending' | 'descending' | 'none' = 'none';
    if (active) {
        ariaSort = ascending ? 'ascending' : 'descending';
    }

    return (
        <th
            className={`${numeric ? 'governance-cell--num ' : ''}governance-th--sortable${active ? ' is-active' : ''}`}
            aria-sort={ariaSort}
        >
            <button
                type='button'
                className='governance-sort-button'
                onClick={() => onSort(column)}
            >
                <span>{label}</span>
                {active && <i className={`icon icon-chevron-${ascending ? 'up' : 'down'}`}/>}
            </button>
        </th>
    );
};

function buildFilters(timeRange?: TimeRange): Array<{value: GovernanceFilter; label: string}> {
    return [
        {value: '', label: 'All'},
        {value: 'unlabelled', label: 'Missing a purpose'},
        {value: 'inactive', label: noPostsLabel(timeRange)},
        {value: 'private', label: 'Private'},
    ];
}

interface ToolbarProps {
    search?: string;
    onSearch?: (value: string) => void;
    filter?: GovernanceFilter;
    onFilter?: (value: GovernanceFilter) => void;
    matching?: number;
    total?: number;
    timeRange?: TimeRange;
}

const Toolbar: React.FC<ToolbarProps> = ({search, onSearch, filter, onFilter, matching, total, timeRange}) => {
    if (!onSearch && !onFilter) {
        return null;
    }
    const narrowed = matching !== undefined && total !== undefined && matching !== total;
    return (
        <div className='governance-toolbar'>
            {onSearch && (
                <input
                    className='governance-search'
                    type='search'
                    value={search ?? ''}
                    placeholder='Search channels and purposes'
                    aria-label='Search channels and purposes'
                    data-testid='governance-search'
                    onChange={(e) => onSearch(e.target.value)}
                />
            )}
            {onFilter && (
                <div
                    className='governance-filters'
                    role='group'
                    aria-label='Filter channels'
                >
                    {buildFilters(timeRange).map((f) => (
                        <button
                            key={f.value || 'all'}
                            type='button'
                            className={`governance-chip${(filter ?? '') === f.value ? ' is-active' : ''}`}
                            aria-pressed={(filter ?? '') === f.value}
                            onClick={() => onFilter(f.value)}
                        >
                            {f.label}
                        </button>
                    ))}
                </div>
            )}
            {narrowed && (
                <span
                    className='governance-matchcount'
                    data-testid='governance-matchcount'
                >
                    {`${matching} of ${total}`}
                </span>
            )}
        </div>
    );
};

// The numbers are up to a day old by design. Saying so is the difference
// between "this channel is quiet" and "we last looked yesterday".
const Freshness: React.FC<{generatedAt?: number}> = ({generatedAt}) => {
    if (!generatedAt) {
        return null;
    }
    const when = new Date(generatedAt).toLocaleString([], {
        year: 'numeric', month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit',
    });
    return (
        <p
            className='governance-freshness'
            data-testid='governance-freshness'
        >
            <FormattedMessage
                id='insights.governance.freshness'
                defaultMessage='Data as of {when} · refreshed daily'
                values={{when}}
            />
        </p>
    );
};

const NotSet: React.FC = () => (
    <span className='governance-notset'>
        <span
            className='governance-notset__dot'
            aria-hidden='true'
        />
        <FormattedMessage
            id='insights.governance.notSet'
            defaultMessage='Not set'
        />
    </span>
);

const ChannelGovernanceListComponent: React.FC<Props> = ({
    items, summary, loading, error, onSelectChannel, sort, ascending, onSort,
    search, onSearch, filter, onFilter, generatedAt, trailingTile, timeRange,
}) => {
    const rows = useMemo(() => items.map((c) => {
        const isPrivate = c.type === 'P';
        return (
            <tr
                key={c.id}
                data-testid={`governance-row-${c.name}`}
                className='governance-row'
                onClick={onSelectChannel ? () => onSelectChannel(c) : undefined}
            >
                <td className='governance-cell governance-cell--name'>
                    <i className={`icon icon-${isPrivate ? 'lock-outline' : 'globe'}`}/>
                    <span>{c.display_name || c.name}</span>
                </td>
                <td className='governance-cell'>
                    {isPrivate ? (
                        <FormattedMessage
                            id='insights.governance.private'
                            defaultMessage='Private'
                        />
                    ) : (
                        <FormattedMessage
                            id='insights.governance.public'
                            defaultMessage='Public'
                        />
                    )}
                </td>
                <td className='governance-cell governance-cell--num'>{c.message_count}</td>
                <td className='governance-cell governance-cell--num'>{c.active_posters}</td>
                <td className='governance-cell governance-cell--num'>{c.member_count}</td>
                <td className='governance-cell'>{formatLastPost(c.last_post_at)}</td>
                <td className='governance-cell'>{formatDate(c.create_at)}</td>
                <td className='governance-cell governance-cell--purpose'>
                    {c.purpose || <NotSet/>}
                </td>
            </tr>
        );
    }), [items, onSelectChannel]);

    // The outer container is always rendered so every state is a descendant
    // of it — callers (and the CT suite) can query one stable root.
    if (error) {
        return (
            <div className='ChannelGovernanceList'>
                <div
                    className='governance-error'
                    data-testid='governance-error'
                >
                    {error}
                </div>
            </div>
        );
    }

    if (loading) {
        return (
            <div className='ChannelGovernanceList'>
                <div
                    className='governance-loading'
                    data-testid='governance-loading'
                >
                    <FormattedMessage
                        id='insights.governance.loading'
                        defaultMessage='Loading channels…'
                    />
                </div>
            </div>
        );
    }

    return (
        <div className='ChannelGovernanceList'>
            {summary && (
                <CoverageSummary
                    summary={summary}
                    trailingTile={trailingTile}
                    timeRange={timeRange}
                />
            )}
            <Freshness generatedAt={generatedAt}/>
            <Toolbar
                search={search}
                onSearch={onSearch}
                filter={filter}
                onFilter={onFilter}
                matching={summary?.matching_channels}
                total={summary?.total_channels}
                timeRange={timeRange}
            />
            {items.length === 0 ? (
                <div
                    className='governance-empty'
                    data-testid='governance-empty'
                >
                    <FormattedMessage
                        id='insights.governance.empty'
                        defaultMessage='No channels to show for this time range.'
                    />
                </div>
            ) : (
                <table className='governance-table'>
                    <thead>
                        <tr>
                            <SortableHeader
                                column='name'
                                sort={sort}
                                ascending={ascending}
                                onSort={onSort}
                                label={<FormattedMessage
                                    id='insights.governance.channel'
                                    defaultMessage='Channel'
                                       />}
                            />
                            <th>
                                <FormattedMessage
                                    id='insights.governance.type'
                                    defaultMessage='Type'
                                />
                            </th>
                            <SortableHeader
                                column='posts'
                                numeric={true}
                                sort={sort}
                                ascending={ascending}
                                onSort={onSort}
                                label={<FormattedMessage
                                    id='insights.governance.posts'
                                    defaultMessage='Posts'
                                       />}
                            />
                            <SortableHeader
                                column='active_users'
                                numeric={true}
                                sort={sort}
                                ascending={ascending}
                                onSort={onSort}
                                label={<FormattedMessage
                                    id='insights.governance.activeUsers'
                                    defaultMessage='Active users'
                                       />}
                            />
                            <SortableHeader
                                column='members'
                                numeric={true}
                                sort={sort}
                                ascending={ascending}
                                onSort={onSort}
                                label={<FormattedMessage
                                    id='insights.governance.members'
                                    defaultMessage='Members'
                                       />}
                            />
                            <SortableHeader
                                column='last_post'
                                sort={sort}
                                ascending={ascending}
                                onSort={onSort}
                                label={<FormattedMessage
                                    id='insights.governance.lastPost'
                                    defaultMessage='Last post'
                                       />}
                            />
                            <SortableHeader
                                column='created'
                                sort={sort}
                                ascending={ascending}
                                onSort={onSort}
                                label={<FormattedMessage
                                    id='insights.governance.created'
                                    defaultMessage='Created'
                                       />}
                            />
                            <th>
                                <FormattedMessage
                                    id='insights.governance.purpose'
                                    defaultMessage='Purpose'
                                />
                            </th>
                        </tr>
                    </thead>
                    <tbody>{rows}</tbody>
                </table>
            )}
        </div>
    );
};

export const ChannelGovernanceList = memo(ChannelGovernanceListComponent);
