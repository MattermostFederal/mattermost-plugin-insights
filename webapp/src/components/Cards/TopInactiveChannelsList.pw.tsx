import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopInactiveChannelsList} from './TopInactiveChannelsList';

const sample = {
    id: 'ch1',
    type: 'O',
    display_name: 'Deserted',
    name: 'deserted',
    last_activity_at: 0,
    participants: ['u1', 'u2', 'u3'],
};

test('shows skeleton when status is loading', async ({mount}) => {
    const component = await mount(
        <TopInactiveChannelsList
            items={[]}
            hasNext={false}
            status='loading'
        />,
    );
    await expect(component.locator('.least-active-channels-loading-container')).toHaveCount(4);
});

test('shows empty state', async ({mount}) => {
    const component = await mount(
        <TopInactiveChannelsList
            items={[]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component).toContainText('Not enough data');

    // Don't be too tight here — the empty-state icon is a globe but the
    // "this channel is private" lock-outline icon is also possible
    // elsewhere in the document.
    await expect(component.locator('.insights-empty-state__icon i.icon-globe')).toHaveCount(1);
});

test('row renders channel-type icon, display name, "No activity", and avatar pill', async ({mount}) => {
    const component = await mount(
        <TopInactiveChannelsList
            items={[sample]}
            hasNext={false}
            status='idle'
        />,
    );
    const row = component.locator('.channel-row').first();
    await expect(row).toBeVisible();

    // Channel-type icon: 'O' → globe
    await expect(row.locator('.channel-display-name i.icon-globe')).toHaveCount(1);

    // Display name
    await expect(row.locator('.display-name')).toContainText('Deserted');

    // "No activity" copy when last_activity_at === 0
    await expect(row.locator('.last-activity')).toContainText('No activity');

    // Avatar pill (3 participants)
    await expect(row.locator('.Avatars .Avatar')).toHaveCount(3);
});

test('private channels render a lock-outline icon', async ({mount}) => {
    const component = await mount(
        <TopInactiveChannelsList
            items={[{...sample, type: 'P'}]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component.locator('.channel-display-name i.icon-lock-outline')).toHaveCount(1);
});

test('rows with non-zero last_activity_at render a relative timestamp prefix', async ({mount}) => {
    const component = await mount(
        <TopInactiveChannelsList
            items={[{...sample, last_activity_at: Date.now() - 86_400_000}]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component.locator('.last-activity')).toContainText('Last activity:');
});
