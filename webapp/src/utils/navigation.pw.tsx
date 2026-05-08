import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopChannelsList} from '../components/Cards/TopChannelsList';

test('clicking a channel row navigates to /{teamName}/channels/{channelName}', async ({mount, page}) => {
    const component = await mount(
        <TopChannelsList
            items={[{id: 'ch1', type: 'O', display_name: 'General', name: 'general', team_id: 't1', message_count: 5}]}
            hasNext={false}
            status='idle'
            teamName='ad-1'
        />,
    );
    await page.evaluate(() => {
        const w = window as unknown as {WebappUtils?: {browserHistory?: {push?: (p: string) => void}}; __captured?: string};
        w.WebappUtils = {
            browserHistory: {
                push: (p: string) => {
                    w.__captured = p;
                },
            },
        };
    });
    await component.locator('.channel-row').first().click();
    const pushed = await page.evaluate(() => (window as unknown as {__captured?: string}).__captured);
    expect(pushed).toBe('/ad-1/channels/general');
});

test('without teamName, clicking a channel row does not navigate', async ({mount, page}) => {
    const component = await mount(
        <TopChannelsList
            items={[{id: 'ch1', type: 'O', display_name: 'General', name: 'general', team_id: 't1', message_count: 5}]}
            hasNext={false}
            status='idle'
        />,
    );
    await page.evaluate(() => {
        const w = window as unknown as {WebappUtils?: {browserHistory?: {push?: (p: string) => void}}; __captured?: string};
        w.__captured = undefined;
        w.WebappUtils = {
            browserHistory: {
                push: (p: string) => {
                    w.__captured = p;
                },
            },
        };
    });
    await component.locator('.channel-row').first().click();
    const pushed = await page.evaluate(() => (window as unknown as {__captured?: string}).__captured);
    expect(pushed).toBeUndefined();
});
