import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopBoardsList} from './TopBoardsList';

const sampleBoard = {
    boardID: 'b1',
    icon: '💬',
    title: 'Project planning',
    activityCount: '12',
    activeUsers: ['u1', 'u2'],
    createdBy: 'u1',
};

test('shows skeleton when status is loading', async ({mount}) => {
    const component = await mount(
        <TopBoardsList
            items={[]}
            hasNext={false}
            status='loading'
        />,
    );
    await expect(component.locator('.top-board-loading-container')).toHaveCount(4);
});

test('shows empty state', async ({mount}) => {
    const component = await mount(
        <TopBoardsList
            items={[]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component).toContainText('Not enough data');
    await expect(component.locator('i.icon-product-boards')).toHaveCount(1);
});

test('shows an error when status is error', async ({mount}) => {
    const component = await mount(
        <TopBoardsList
            items={[]}
            hasNext={false}
            status='error'
            error='boom'
        />,
    );
    await expect(component).toContainText('boom');
});

test('renders a row per board with title and update count', async ({mount}) => {
    const component = await mount(
        <TopBoardsList
            items={[
                sampleBoard,
                {boardID: 'b2', icon: '📋', title: 'Roadmap', activityCount: '6', activeUsers: ['u1'], createdBy: 'u1'},
            ]}
            hasNext={false}
            status='idle'
        />,
    );
    const rows = component.locator('.board-item');
    await expect(rows).toHaveCount(2);
    await expect(rows.nth(0)).toContainText('Project planning');
    await expect(rows.nth(0)).toContainText('12');
    await expect(rows.nth(1)).toContainText('Roadmap');
    await expect(rows.nth(1)).toContainText('6');
});

test('handles activeUsers as a comma-joined string (deprecated payload shape)', async ({mount}) => {
    // The deprecated TopBoard JSON sometimes carried activeUsers as a CSV
    // string instead of an array (MM-49023). Treat both shapes as
    // equivalent.
    const component = await mount(
        <TopBoardsList
            items={[{...sampleBoard, activeUsers: 'u1,u2,u3' as unknown as string[]}]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component).toContainText('Project planning');
});
