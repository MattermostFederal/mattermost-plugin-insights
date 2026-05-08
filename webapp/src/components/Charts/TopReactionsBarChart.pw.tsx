import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopReactionsBarChart} from './TopReactionsBarChart';

test('renders one bar per reaction with emoji and count', async ({mount}) => {
    const component = await mount(
        <TopReactionsBarChart
            reactions={[
                {emoji_name: 'smile', count: 10},
                {emoji_name: 'joy', count: 6},
                {emoji_name: 'fire', count: 4},
            ]}
        />,
    );
    await expect(component.locator('.bar-chart-entry')).toHaveCount(3);
    await expect(component).toContainText('10');
    await expect(component).toContainText('6');
    await expect(component).toContainText('4');
});

test('the tallest bar is rendered at full max-height', async ({mount}) => {
    const component = await mount(
        <TopReactionsBarChart
            reactions={[
                {emoji_name: 'smile', count: 10},
                {emoji_name: 'joy', count: 5},
            ]}
        />,
    );
    const bars = component.locator('.bar-chart-data');
    await expect(bars).toHaveCount(2);
    const firstHeight = await bars.nth(0).evaluate((el) => (el as HTMLElement).style.height);
    const secondHeight = await bars.nth(1).evaluate((el) => (el as HTMLElement).style.height);
    expect(firstHeight).toBe('156px');
    expect(secondHeight).toBe('78px');
});

test('renders an emoji image with the host system-emoji URL pattern', async ({mount, page}) => {
    // Stub the static emoji URL so the <img> stays mounted (the production
    // code falls back to a `:name:` span when the image errors).
    await page.route('**/static/emoji/*.png', (route) =>
        route.fulfill({
            status: 200,
            contentType: 'image/png',
            body: Buffer.from(
                'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=',
                'base64',
            ),
        }),
    );
    const component = await mount(
        <TopReactionsBarChart reactions={[{emoji_name: 'smile', count: 1}]}/>,
    );
    const img = component.locator('img.emoticon');
    await expect(img).toHaveCount(1);
    await expect(img).toHaveAttribute('src', /\/static\/emoji\/[0-9a-f-]+\.png/);
});

test('falls back to `:name:` text when the image fails to load', async ({mount, page}) => {
    await page.route('**/static/emoji/*.png', (route) => route.fulfill({status: 404}));
    await page.route('**/api/v4/emoji/name/**', (route) => route.fulfill({status: 404}));
    const component = await mount(
        <TopReactionsBarChart reactions={[{emoji_name: 'made_up_name_qq', count: 1}]}/>,
    );
    await expect(component).toContainText(':made_up_name_qq:');
});
