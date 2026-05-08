import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {ChannelActionsMenu} from './ChannelActionsMenu';

const sampleChannel = {
    id: 'ch1',
    type: 'O',
    display_name: 'Random',
    name: 'random',
    last_activity_at: 0,
    participants: ['u1', 'u2'],
};

test('renders the kebab toggle and opens the menu on click', async ({mount}) => {
    const component = await mount(
        <ChannelActionsMenu
            channel={sampleChannel}
            teamName='ad-1'
            currentUserId='u1'
            isMember={true}
        />,
    );
    await expect(component.locator('.channel-action__toggle')).toHaveCount(1);
    await component.locator('.channel-action__toggle').click();
    await expect(component.locator('.channel-action__menu')).toHaveCount(1);
    await expect(component).toContainText('Leave channel');
    await expect(component).toContainText('Copy link');
});

test('does not show "Leave channel" when the user is not a member', async ({mount}) => {
    const component = await mount(
        <ChannelActionsMenu
            channel={sampleChannel}
            teamName='ad-1'
            currentUserId='u1'
            isMember={false}
        />,
    );
    await component.locator('.channel-action__toggle').click();
    await expect(component).not.toContainText('Leave channel');
    await expect(component).toContainText('Copy link');
});

test('does not show "Leave channel" for the default channel (town-square)', async ({mount}) => {
    const component = await mount(
        <ChannelActionsMenu
            channel={{...sampleChannel, name: 'town-square'}}
            teamName='ad-1'
            currentUserId='u1'
            isMember={true}
        />,
    );
    await component.locator('.channel-action__toggle').click();
    await expect(component).not.toContainText('Leave channel');
});

test('clicking "Leave channel" on a public channel calls the leave endpoint', async ({mount, page}) => {
    let leaveCalled = false;
    await page.route('**/api/v4/channels/ch1/members/u1', (route) => {
        if (route.request().method() === 'DELETE') {
            leaveCalled = true;
            route.fulfill({status: 200, contentType: 'application/json', body: JSON.stringify({status: 'OK'})});
            return;
        }
        route.fallback();
    });
    const component = await mount(
        <ChannelActionsMenu
            channel={sampleChannel}
            teamName='ad-1'
            currentUserId='u1'
            isMember={true}
        />,
    );
    await component.locator('.channel-action__toggle').click();
    await component.locator('.channel-action__menu-item--danger').click();
    await expect.poll(() => leaveCalled).toBe(true);
});

test('clicking "Leave channel" on a private channel opens a confirmation modal', async ({mount, page}) => {
    const component = await mount(
        <ChannelActionsMenu
            channel={{...sampleChannel, type: 'P', display_name: 'Secret'}}
            teamName='ad-1'
            currentUserId='u1'
            isMember={true}
        />,
    );
    await component.locator('.channel-action__toggle').click();
    await component.locator('.channel-action__menu-item--danger').click();

    // The bootstrap modal portals to body — assert at page level.
    await expect(page.getByText('Leave private channel').first()).toBeVisible();
    await expect(page.getByText('Yes, leave channel')).toBeVisible();
});
