import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopBoardsTable} from './TopBoardsTable';

test('hits the user route, renders rows, and paginates', async ({mount, page}) => {
    let pageRequested = -1;
    await page.route('**/plugins/insights/api/v1/users/me/top/boards**', (route) => {
        const url = new URL(route.request().url());
        const reqPage = Number(url.searchParams.get('page') || '0');
        pageRequested = reqPage;
        const items = reqPage === 0 ?
            [
                {boardID: 'b1', icon: '💬', title: 'Project planning', activityCount: '12', activeUsers: ['u1'], createdBy: 'u1'},
                {boardID: 'b2', icon: '📋', title: 'Roadmap', activityCount: '6', activeUsers: ['u1'], createdBy: 'u1'},
            ] :
            [{boardID: 'b3', icon: '🎯', title: 'Goals', activityCount: '4', activeUsers: ['u1'], createdBy: 'u1'}];
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({has_next: reqPage === 0, items}),
        });
    });

    const component = await mount(
        <TopBoardsTable
            scope='my'
            timeRange='7_day'
            teamId='team1'
        />,
    );

    await expect(component.locator('.DataGrid_row')).toHaveCount(2);
    await expect(component).toContainText('Project planning');
    await expect(component).toContainText('Roadmap');
    expect(pageRequested).toBe(0);

    await component.locator('button.next').click();
    await expect(component.locator('.DataGrid_row')).toHaveCount(1);
    await expect(component).toContainText('Goals');
    expect(pageRequested).toBe(1);
});

test('team scope hits the team route', async ({mount, page}) => {
    let calledPath = '';
    await page.route('**/plugins/insights/api/v1/teams/**/top/boards**', (route) => {
        calledPath = new URL(route.request().url()).pathname;
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({has_next: false, items: []}),
        });
    });
    await mount(
        <TopBoardsTable
            scope='team'
            timeRange='today'
            teamId='team42'
        />,
    );
    await expect.poll(() => calledPath).toContain('/plugins/insights/api/v1/teams/team42/top/boards');
});
