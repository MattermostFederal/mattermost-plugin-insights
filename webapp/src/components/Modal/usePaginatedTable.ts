// Local pagination state for the modal tables. The deprecated tables stored
// their pages in Redux, but the modal view fetches `per_page=10` while the
// card view fetches `per_page=5`; sharing one cache would force one or the
// other to re-fetch every time. Local state keeps things simple.

import {useCallback, useEffect, useState} from 'react';

const PER_PAGE = 10;

interface PageState<T> {
    items: T[];
    page: number;
    hasNext: boolean;
    loading: boolean;
    error?: string;
    extra?: unknown;
}

export interface PaginatedTableState<T> extends PageState<T> {
    perPage: number;
    nextPage: () => void;
    previousPage: () => void;
}

export function usePaginatedTable<T, Resp extends {has_next: boolean; items: T[]}>(
    fetcher: (page: number, perPage: number) => Promise<Resp>,
    deps: readonly unknown[],
    onResponse?: (resp: Resp) => void,
): PaginatedTableState<T> {
    const [state, setState] = useState<PageState<T>>({items: [], page: 0, hasNext: false, loading: true});

    const load = useCallback(async (page: number) => {
        setState((s) => ({...s, loading: true, error: undefined}));
        try {
            const resp = await fetcher(page, PER_PAGE);
            setState({items: resp.items, page, hasNext: resp.has_next, loading: false});
            if (onResponse) {
                onResponse(resp);
            }
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to load';
            setState((s) => ({...s, loading: false, error: message}));
        }
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, deps);

    useEffect(() => {
        load(0);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, deps);

    const nextPage = useCallback(() => load(state.page + 1), [load, state.page]);
    const previousPage = useCallback(() => load(Math.max(0, state.page - 1)), [load, state.page]);

    return {
        ...state,
        perPage: PER_PAGE,
        nextPage,
        previousPage,
    };
}
