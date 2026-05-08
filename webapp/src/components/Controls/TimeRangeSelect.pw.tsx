import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TimeRangeSelect} from './TimeRangeSelect';

test('renders the current time-frame label', async ({mount}) => {
    const component = await mount(
        <TimeRangeSelect
            value='7_day'
            onChange={() => undefined}
        />,
    );
    await expect(component).toContainText('Last 7 days');
});

test('opens menu and switches to a new range', async ({mount}) => {
    let chosen: string | undefined;
    const component = await mount(
        <TimeRangeSelect
            value='7_day'
            onChange={(r) => {
                chosen = r;
            }}
        />,
    );
    await component.locator('.insights-time-range__control').click();
    await component.page().getByText('Today', {exact: true}).click();
    expect(chosen).toBe('today');
});
