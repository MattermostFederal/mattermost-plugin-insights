import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopPlaybooksTable} from './TopPlaybooksTable';

test('hits the user route, renders rows, and paginates', async ({mount, page}) => {
    let pageRequested = -1;
    await page.route('**/plugins/insights/api/v1/users/me/top/playbooks**', (route) => {
        const url = new URL(route.request().url());
        const reqPage = Number(url.searchParams.get('page') || '0');
        pageRequested = reqPage;
        const items = reqPage === 0
            ? [
                {playbook_id: 'pb1', num_runs: 10, title: 'Incident response', last_run_at: 1_700_000_000_000},
                {playbook_id: 'pb2', num_runs: 6, title: 'Onboarding', last_run_at: 1_700_000_000_000},
            ]
            : [{playbook_id: 'pb3', num_runs: 4, title: 'Release', last_run_at: 1_700_000_000_000}];
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({has_next: reqPage === 0, items}),
        });
    });

    const component = await mount(
        <TopPlaybooksTable
            scope='my'
            timeRange='7_day'
            teamId='team1'
        />,
    );

    await expect(component.locator('.DataGrid_row')).toHaveCount(2);
    await expect(component).toContainText('Incident response');
    await expect(component).toContainText('Onboarding');
    expect(pageRequested).toBe(0);

    await component.locator('button.next').click();
    await expect(component.locator('.DataGrid_row')).toHaveCount(1);
    await expect(component).toContainText('Release');
    expect(pageRequested).toBe(1);
});

test('team scope hits the team route', async ({mount, page}) => {
    let calledPath = '';
    await page.route('**/plugins/insights/api/v1/teams/**/top/playbooks**', (route) => {
        calledPath = new URL(route.request().url()).pathname;
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({has_next: false, items: []}),
        });
    });
    await mount(
        <TopPlaybooksTable
            scope='team'
            timeRange='today'
            teamId='team42'
        />,
    );
    await expect.poll(() => calledPath).toContain('/plugins/insights/api/v1/teams/team42/top/playbooks');
});
