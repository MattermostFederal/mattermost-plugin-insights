import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopDMsList} from './TopDMsList';

const sampleDM = {
    post_count: 10,
    outgoing_message_count: 6,
    second_participant: {
        id: 'partner1',
        username: 'partner.one',
        first_name: 'Partner',
        last_name: 'One',
        nickname: 'p1',
        last_picture_update: 0,
        position: 'Director',
    },
};

test('shows skeleton when status is loading', async ({mount}) => {
    const component = await mount(
        <TopDMsList
            items={[]}
            hasNext={false}
            status='loading'
        />,
    );
    await expect(component.locator('.dms-loading-container')).toHaveCount(5);
});

test('shows empty state', async ({mount}) => {
    const component = await mount(
        <TopDMsList
            items={[]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component).toContainText('Not enough data');
    await expect(component.locator('i.icon-account-multiple-outline')).toHaveCount(1);
});

test('row renders avatar, partner name, position, count, and proportional bar', async ({mount}) => {
    const component = await mount(
        <TopDMsList
            items={[sampleDM]}
            hasNext={false}
            status='idle'
        />,
    );
    const row = component.locator('.top-dms-item').first();
    await expect(row).toBeVisible();

    // Avatar
    await expect(row.locator('.Avatar')).toHaveCount(1);

    // Partner display name + position
    await expect(row.locator('.dm-name')).toContainText('Partner One');
    await expect(row.locator('.dm-role')).toContainText('Director');

    // Message count + horizontal bar
    await expect(row.locator('.message-count')).toContainText('10');
    await expect(row.locator('.horizontal-bar')).toHaveCount(1);
});

test('falls back to username when no first/last name', async ({mount}) => {
    const component = await mount(
        <TopDMsList
            items={[{...sampleDM, second_participant: {...sampleDM.second_participant, first_name: '', last_name: ''}}]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component.locator('.dm-name')).toContainText('partner.one');
});

test('omits position pill when participant has no position', async ({mount}) => {
    const component = await mount(
        <TopDMsList
            items={[{...sampleDM, second_participant: {...sampleDM.second_participant, position: ''}}]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component.locator('.dm-role')).toHaveCount(0);
});

test('top DM gets full-width bar; second DM gets a smaller bar', async ({mount}) => {
    const component = await mount(
        <TopDMsList
            items={[
                sampleDM, // post_count = 10
                {...sampleDM, second_participant: {...sampleDM.second_participant, id: 'p2'}, post_count: 5},
            ]}
            hasNext={false}
            status='idle'
        />,
    );
    const bars = component.locator('.horizontal-bar');
    await expect(bars).toHaveCount(2);
    const firstFlex = await bars.nth(0).evaluate((el) => (el as HTMLElement).style.flex);
    const secondFlex = await bars.nth(1).evaluate((el) => (el as HTMLElement).style.flex);
    expect(firstFlex).toContain('1 ');
    expect(secondFlex).toContain('0.5 ');
});
