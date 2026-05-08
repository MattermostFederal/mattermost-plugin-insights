import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {InsightsModal} from './InsightsModal';

test.beforeEach(async ({page}) => {
    // Stub the insights API endpoints so the table fetcher resolves.
    await page.route('**/plugins/insights/api/v1/**', (route) =>
        route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({has_next: false, items: []}),
        }),
    );
});

test('renders title and subtitle', async ({mount}) => {
    const component = await mount(
        <InsightsModal
            show={true}
            onExited={() => undefined}
            widgetType='topReactions'
            title='Top Reactions'
            subtitle='The most used emoji reactions'
            scope='my'
            timeRange='7_day'
            teamId='team1'
        />,
    );
    await expect(component.page().locator('#insightsModalTitle')).toContainText('Top Reactions');
    await expect(component.page().locator('.subtitle')).toContainText('The most used emoji reactions');
});

test('renders a time-range dropdown in the header', async ({mount}) => {
    const component = await mount(
        <InsightsModal
            show={true}
            onExited={() => undefined}
            widgetType='topReactions'
            title='Top Reactions'
            subtitle='subtitle'
            scope='my'
            timeRange='7_day'
            teamId='team1'
        />,
    );
    await expect(component.page().locator('.insights-time-range__control')).toHaveCount(1);
});
