import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {NewMembersTable} from './NewMembersTable';

const daysAgo = (days: number): number => {
    const date = new Date();
    date.setDate(date.getDate() - days);
    return date.getTime();
};

const members = [
    {
        id: 'u1',
        username: 'grace',
        first_name: 'Grace',
        last_name: 'Hopper',
        position: 'Principal Engineer',
        nickname: '',
        create_at: daysAgo(2),
    },
    {
        id: 'u2',
        username: 'bob',
        first_name: '',
        last_name: '',
        position: '',
        nickname: '',
        create_at: daysAgo(1),
    },
];

async function routeMembers(page: import('@playwright/test').Page) {
    await page.route('**/plugins/insights/api/v1/teams/*/top/team_members**', (route) => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({has_next: false, items: members, total_count: 2}),
    }));
    await page.route('**/static/emoji/*.png', (route) => route.fulfill({status: 404}));
    await page.route('**/api/v4/users/*/image**', (route) => route.fulfill({status: 404}));
}

// The card that used to show faces is gone, so the modal has to be the place
// you can recognise someone — a name-only list would have lost that.
test('shows an avatar and username alongside each name', async ({mount, page}) => {
    await routeMembers(page);
    const component = await mount(
        <NewMembersTable
            scope='team'
            timeRange='7_day'
            teamId='team1'
        />,
    );

    await expect(component.locator('.DataGrid_row')).toHaveCount(2);
    await expect(component).toContainText('Grace Hopper');
    await expect(component).toContainText('@grace');
    await expect(component.locator('.new-members-cell .Avatar').first()).toBeVisible();
});

test('falls back to the username when no full name is set', async ({mount, page}) => {
    await routeMembers(page);
    const component = await mount(
        <NewMembersTable
            scope='team'
            timeRange='7_day'
            teamId='team1'
        />,
    );
    await expect(component.locator('.DataGrid_row').nth(1)).toContainText('bob');
});

test('carries the Say hello action the card used to show', async ({mount, page}) => {
    await routeMembers(page);
    const component = await mount(
        <NewMembersTable
            scope='team'
            timeRange='7_day'
            teamId='team1'
        />,
    );
    await expect(component.locator('.new-members-cell__hello').first()).toContainText('Say hello');
});

test('renders position and a relative join date', async ({mount, page}) => {
    await routeMembers(page);
    const component = await mount(
        <NewMembersTable
            scope='team'
            timeRange='7_day'
            teamId='team1'
        />,
    );
    await expect(component).toContainText('Principal Engineer');
    await expect(component).toContainText('2 days ago');
});

// Calendar words are capitalised like the cell values they are, and counted
// in whole days rather than elapsed hours — joining at 23:00 yesterday must
// not read as "Today" the next morning.
test('capitalises Yesterday and counts calendar days', async ({mount, page}) => {
    await routeMembers(page);
    const component = await mount(
        <NewMembersTable
            scope='team'
            timeRange='7_day'
            teamId='team1'
        />,
    );
    await expect(component).toContainText('Yesterday');
    await expect(component).not.toContainText('yesterday');
    await expect(component).not.toContainText('today');
});
