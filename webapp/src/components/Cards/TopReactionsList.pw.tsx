import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopReactionsList} from './TopReactionsList';

test('renders nothing when status is idle and items are empty', async ({mount}) => {
    const component = await mount(
        <TopReactionsList
            items={[]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component).toContainText('Not enough data');
    await expect(component.locator('i.icon-emoticon-outline')).toHaveCount(1);
});

test('shows a skeleton when status is loading', async ({mount}) => {
    const component = await mount(
        <TopReactionsList
            items={[]}
            hasNext={false}
            status='loading'
        />,
    );
    await expect(component.locator('.bar-chart-entry')).toHaveCount(5);
});

test('shows an error when status is error', async ({mount}) => {
    const component = await mount(
        <TopReactionsList
            items={[]}
            hasNext={false}
            status='error'
            error='boom'
        />,
    );
    await expect(component).toContainText('boom');
});

test('renders items in order with rank, emoji name, and count', async ({mount}) => {
    const component = await mount(
        <TopReactionsList
            items={[
                {emoji_name: '100', count: 6},
                {emoji_name: 'joy', count: 5},
                {emoji_name: 'smile', count: 4},
            ]}
            hasNext={false}
            status='idle'
        />,
    );
    const rows = component.locator('li');
    await expect(rows).toHaveCount(3);
    await expect(rows.nth(0)).toContainText('100');
    await expect(rows.nth(0)).toContainText('6');
    await expect(rows.nth(1)).toContainText('joy');
    await expect(rows.nth(1)).toContainText('5');
    await expect(rows.nth(2)).toContainText('smile');
    await expect(rows.nth(2)).toContainText('4');
});
