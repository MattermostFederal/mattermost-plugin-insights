import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {InsightCard} from './InsightCard';

test('renders title and children', async ({mount}) => {
    const component = await mount(
        <InsightCard title='Top Reactions'>
            <p>{'body content'}</p>
        </InsightCard>,
    );
    await expect(component).toContainText('Top Reactions');
    await expect(component).toContainText('body content');
});

test('renders subtitle when provided', async ({mount}) => {
    const component = await mount(
        <InsightCard
            title='Top Reactions'
            subtitle='Most used reactions in your team'
        >
            <p>{'body'}</p>
        </InsightCard>,
    );
    await expect(component).toContainText('Most used reactions in your team');
});

test('omits chevron button when onClick is not provided', async ({mount}) => {
    const component = await mount(
        <InsightCard title='Top Reactions'>
            <p>{'body'}</p>
        </InsightCard>,
    );
    await expect(component.locator('button.insights-card__expand')).toHaveCount(0);
});

test('renders chevron button and fires onClick when handler is provided', async ({mount}) => {
    let clicked = 0;
    const component = await mount(
        <InsightCard
            title='Top Reactions'
            onClick={() => {
                clicked += 1;
            }}
        >
            <p>{'body'}</p>
        </InsightCard>,
    );
    const button = component.locator('button.insights-card__expand');
    await expect(button).toHaveCount(1);
    await button.click();
    expect(clicked).toBe(1);
});

test('clicking the header (title row) also fires onClick when provided', async ({mount}) => {
    let clicked = 0;
    const component = await mount(
        <InsightCard
            title='Top Reactions'
            onClick={() => {
                clicked += 1;
            }}
        >
            <p>{'body'}</p>
        </InsightCard>,
    );
    await component.locator('.insights-card__header').click();
    expect(clicked).toBe(1);
});
