import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopPlaybooksList} from './TopPlaybooksList';

const samplePlaybook = {
    playbook_id: 'pb1',
    num_runs: 12,
    title: 'Incident response',
    last_run_at: 1_700_000_000_000,
};

test('shows skeleton when status is loading', async ({mount}) => {
    const component = await mount(
        <TopPlaybooksList
            items={[]}
            hasNext={false}
            status='loading'
        />,
    );
    await expect(component.locator('.top-playbooks-loading-container')).toHaveCount(3);
});

test('shows empty state', async ({mount}) => {
    const component = await mount(
        <TopPlaybooksList
            items={[]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component).toContainText('Not enough data');
    await expect(component.locator('i.icon-product-playbooks')).toHaveCount(1);
});

test('shows an error when status is error', async ({mount}) => {
    const component = await mount(
        <TopPlaybooksList
            items={[]}
            hasNext={false}
            status='error'
            error='boom'
        />,
    );
    await expect(component).toContainText('boom');
});

test('renders a row per playbook with title and run count', async ({mount}) => {
    const component = await mount(
        <TopPlaybooksList
            items={[
                samplePlaybook,
                {playbook_id: 'pb2', num_runs: 6, title: 'Onboarding', last_run_at: 1_700_000_000_000},
            ]}
            hasNext={false}
            status='idle'
        />,
    );
    const rows = component.locator('.playbook-item');
    await expect(rows).toHaveCount(2);
    await expect(rows.nth(0)).toContainText('Incident response');
    await expect(rows.nth(0)).toContainText('12');
    await expect(rows.nth(1)).toContainText('Onboarding');
    await expect(rows.nth(1)).toContainText('6');
});
