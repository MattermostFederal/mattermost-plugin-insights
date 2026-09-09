import {expect, test} from '@playwright/test';

import {nextSortState} from './sorting';

test('clicking the active column flips direction', () => {
    expect(nextSortState({sort: 'posts', ascending: false}, 'posts')).toEqual({sort: 'posts', ascending: true});
    expect(nextSortState({sort: 'posts', ascending: true}, 'posts')).toEqual({sort: 'posts', ascending: false});
});

test('count columns open at the largest value', () => {
    expect(nextSortState({sort: 'name', ascending: true}, 'members')).toEqual({sort: 'members', ascending: false});
    expect(nextSortState({sort: 'name', ascending: true}, 'last_post')).toEqual({sort: 'last_post', ascending: false});
});

// Opening the Channel column descending puts Z first and points the chevron
// down while the rows read A→Z, which is how "clicking Channel does nothing"
// got reported in the first place.
test('the Channel column opens A to Z and still toggles', () => {
    const first = nextSortState({sort: 'last_post', ascending: false}, 'name');
    expect(first).toEqual({sort: 'name', ascending: true});
    expect(nextSortState(first, 'name')).toEqual({sort: 'name', ascending: false});
});
