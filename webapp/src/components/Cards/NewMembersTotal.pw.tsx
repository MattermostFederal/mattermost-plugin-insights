import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {NewMembersTotal} from './NewMembersTotal';

test('renders the total count and 7-day time range info', async ({mount}) => {
    const component = await mount(
        <NewMembersTotal
            total={42}
            timeRange='7_day'
            openInsightsModal={() => undefined}
        />,
    );
    await expect(component).toContainText('42');
    await expect(component).toContainText('Joined the team in the last 7 days');
});

test('renders today copy for the today time range', async ({mount}) => {
    const component = await mount(
        <NewMembersTotal
            total={5}
            timeRange='today'
            openInsightsModal={() => undefined}
        />,
    );
    await expect(component).toContainText('Joined the team today');
});

test('renders 28-day copy for the 28_day time range', async ({mount}) => {
    const component = await mount(
        <NewMembersTotal
            total={100}
            timeRange='28_day'
            openInsightsModal={() => undefined}
        />,
    );
    await expect(component).toContainText('Joined the team in the last 28 days');
});

test('clicking "See all" calls openInsightsModal and stops propagation', async ({mount}) => {
    let opened = 0;
    let outerClicks = 0;
    const component = await mount(
        <div
            onClick={() => {
            outerClicks += 1;
        }}
        >
            <NewMembersTotal
                total={42}
                timeRange='7_day'
                openInsightsModal={() => {
                    opened += 1;
                }}
            />
        </div>,
    );
    await component.locator('.see-all-button').click();
    expect(opened).toBe(1);
    expect(outerClicks).toBe(0);
});
