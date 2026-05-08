import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopChannelsList} from './TopChannelsList';

const sampleChannel = {
    id: 'ch1',
    type: 'O',
    display_name: 'General',
    name: 'general',
    team_id: 't1',
    message_count: 42,
};

test('shows skeleton when status is loading', async ({mount}) => {
    const component = await mount(
        <TopChannelsList
            items={[]}
            hasNext={false}
            status='loading'
        />,
    );
    await expect(component.locator('.top-channel-loading-row')).toHaveCount(5);
});

test('shows empty state', async ({mount}) => {
    const component = await mount(
        <TopChannelsList
            items={[]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component).toContainText('Not enough data');
    await expect(component.locator('.insights-empty-state__icon i.icon-globe')).toHaveCount(1);
});

test('row renders channel-type icon, display name, count, and proportional bar', async ({mount}) => {
    const component = await mount(
        <TopChannelsList
            items={[sampleChannel]}
            hasNext={false}
            status='idle'
        />,
    );
    const row = component.locator('.channel-row').first();
    await expect(row).toBeVisible();

    // Public channel: globe icon
    await expect(row.locator('.channel-display-name i.icon-globe')).toHaveCount(1);

    // Display name + count
    await expect(row.locator('.display-name')).toContainText('General');
    await expect(row.locator('.message-count')).toContainText('42');

    // Horizontal bar present
    await expect(row.locator('.horizontal-bar')).toHaveCount(1);
});

test('private channels render a lock-outline icon', async ({mount}) => {
    const component = await mount(
        <TopChannelsList
            items={[{...sampleChannel, type: 'P'}]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component.locator('.channel-display-name i.icon-lock-outline')).toHaveCount(1);
});

test('top channel gets full-width bar; second channel gets a proportional bar', async ({mount}) => {
    const component = await mount(
        <TopChannelsList
            items={[
                sampleChannel, // 42
                {...sampleChannel, id: 'ch2', name: 'random', message_count: 21}, // half
            ]}
            hasNext={false}
            status='idle'
        />,
    );
    const bars = component.locator('.horizontal-bar');
    await expect(bars).toHaveCount(2);
    const firstFlex = await bars.nth(0).evaluate((el) => (el as HTMLElement).style.flex);
    const secondFlex = await bars.nth(1).evaluate((el) => (el as HTMLElement).style.flex);

    // 0.8 * (42/42) = 0.8 ; 0.8 * (21/42) = 0.4
    expect(firstFlex).toContain('0.8 ');
    expect(secondFlex).toContain('0.4 ');
});
