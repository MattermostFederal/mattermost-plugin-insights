import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {WidgetEmptyState} from './WidgetEmptyState';

test('renders default message and icon class', async ({mount}) => {
    const component = await mount(
        <WidgetEmptyState icon='emoticon-outline'/>,
    );
    await expect(component).toContainText('Not enough data yet');
    await expect(component.locator('i.icon-emoticon-outline')).toHaveCount(1);
});

test('renders a custom message when supplied', async ({mount}) => {
    const component = await mount(
        <WidgetEmptyState
            icon='message-text-outline'
            message='No threads in this window'
        />,
    );
    await expect(component).toContainText('No threads in this window');
});
