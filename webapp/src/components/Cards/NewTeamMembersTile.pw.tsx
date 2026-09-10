import {expect, test} from '@playwright/experimental-ct-react';
import manifest from 'manifest';
import React from 'react';

import {NewTeamMembersTile} from './NewTeamMembersTile';

import reducer from '../../redux/reducer';
import type {NewTeamMembersEntry} from '../../redux/types';

const stateWith = (entry: NewTeamMembersEntry) => {
    const pluginState = reducer(undefined, {type: ''});
    return {
        [`plugins-${manifest.id}`]: {
            ...pluginState,
            newTeamMembers: {team: {team1: {'7_day': entry}}},
        },
    };
};

test('shows an unavailable state when loading members fails without retained data', async ({mount}) => {
    const component = await mount(
        <NewTeamMembersTile
            teamId='team1'
            timeRange='7_day'
        />,
        {hooksConfig: {state: stateWith({items: [], hasNext: false, status: 'error', totalCount: 0})}},
    );

    await expect(component).toContainText('—');
    await expect(component).toContainText('Failed to load new members');
});

test('marks retained member data as stale after a refresh fails', async ({mount}) => {
    const component = await mount(
        <NewTeamMembersTile
            teamId='team1'
            timeRange='7_day'
        />,
        {hooksConfig: {state: stateWith({items: [], hasNext: false, status: 'error', totalCount: 7, fetchedAt: 1})}},
    );

    await expect(component).toContainText('7');
    await expect(component).toContainText('Stale data');
});
