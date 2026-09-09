// Sort-state transition for the channel-governance table. It lives apart from
// the container so it can be tested without mounting Redux and the client:
// which direction a column opens in is a rule, not a rendering detail.

export type SortColumn = 'posts' | 'active_users' | 'members' | 'last_post' | 'created' | 'name';

export interface SortState {
    sort: SortColumn;
    ascending: boolean;
}

// Counts open at the largest — "which channels are busiest" is the usual first
// question. Channel opens A→Z, because a descending alphabet is not what
// anyone means by sorting a name column.
export function opensAscending(column: SortColumn): boolean {
    return column === 'name';
}

// Clicking the active column flips direction; clicking a new one adopts that
// column's natural opening direction.
export function nextSortState(current: SortState, column: SortColumn): SortState {
    if (column === current.sort) {
        return {sort: column, ascending: !current.ascending};
    }
    return {sort: column, ascending: opensAscending(column)};
}
