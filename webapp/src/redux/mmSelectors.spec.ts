import {expect, test} from '@playwright/test';

import {getEffectiveTeamId} from './mmSelectors';

test('waits for team membership state before choosing a fallback team', () => {
    const state = {
        entities: {
            teams: {
                teams: {teamB: {}, teamA: {}},
            },
        },
    };

    expect(getEffectiveTeamId(state)).toBe('');
});

test('chooses the sorted first team present in teams and myMembers', () => {
    const state = {
        entities: {
            teams: {
                teams: {teamC: {}, teamB: {}, teamA: {}},
                myMembers: {teamC: {}, teamA: {}},
            },
        },
    };

    expect(getEffectiveTeamId(state)).toBe('teamA');
});

test('returns no fallback when no loaded team is joined', () => {
    const state = {
        entities: {
            teams: {
                teams: {teamA: {}},
                myMembers: {teamB: {}},
            },
        },
    };

    expect(getEffectiveTeamId(state)).toBe('');
});
