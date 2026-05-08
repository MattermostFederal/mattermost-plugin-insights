import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopReactionsTable} from './TopReactionsTable';

test('hits the user route, renders rows, and paginates', async ({mount, page}) => {
    let pageRequested = -1;
    await page.route('**/plugins/insights/api/v1/users/me/top/reactions**', (route) => {
        const url = new URL(route.request().url());
        const reqPage = Number(url.searchParams.get('page') || '0');
        pageRequested = reqPage;
        const items = reqPage === 0
            ? [
                {emoji_name: 'smile', count: 10},
                {emoji_name: 'joy', count: 6},
            ]
            : [{emoji_name: 'fire', count: 4}];
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({has_next: reqPage === 0, items}),
        });
    });
    await page.route('**/static/emoji/*.png', (route) => route.fulfill({status: 404}));

    const component = await mount(
        <TopReactionsTable
            scope='my'
            timeRange='7_day'
            teamId='team1'
        />,
    );

    await expect(component.locator('.DataGrid_row')).toHaveCount(2);
    await expect(component).toContainText('smile');
    await expect(component).toContainText('joy');
    expect(pageRequested).toBe(0);

    await component.locator('button.next').click();
    await expect(component.locator('.DataGrid_row')).toHaveCount(1);
    await expect(component).toContainText('fire');
    expect(pageRequested).toBe(1);
});
