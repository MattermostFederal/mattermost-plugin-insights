import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {ChannelGovernanceList} from './ChannelGovernanceList';

import type {ChannelActivity, ChannelGovernanceSummary} from '../../../types';

const busy: ChannelActivity = {
    id: 'ch0',
    type: 'O',
    display_name: 'Platform Standup',
    name: 'platform-standup',
    purpose: 'Daily standup for the platform ART',
    header: 'Links here',
    create_at: 1710000000000,
    last_post_at: 1757000000000,
    last_post_in_window: 1757000000000,
    message_count: 1284,
    active_posters: 47,
    member_count: 62,
};

const abandoned: ChannelActivity = {
    id: 'ch1',
    type: 'O',
    display_name: 'Project Halcyon',
    name: 'project-halcyon',
    purpose: '',
    header: '',
    create_at: 1739750000000,
    last_post_at: 0,
    last_post_in_window: 0,
    message_count: 0,
    active_posters: 0,
    member_count: 61,
};

const priv: ChannelActivity = {
    ...busy,
    id: 'ch2',
    type: 'P',
    display_name: 'Platform Leads',
    name: 'platform-leads',
    message_count: 612,
    active_posters: 9,
    member_count: 11,
};

const summary: ChannelGovernanceSummary = {
    total_channels: 412,
    active_channels: 168,
    with_purpose: 38,
    with_header: 12,
};

test('renders a row per channel with all governance columns', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy, abandoned]}
            summary={summary}
        />,
    );

    await expect(component).toContainText('Platform Standup');
    await expect(component).toContainText('1284');
    await expect(component).toContainText('47');
    await expect(component).toContainText('62');
    await expect(component).toContainText('Daily standup for the platform ART');
});

// The reason the table exists: a channel with members and no posts must be
// visible, not filtered out as empty.
test('shows channels with no activity alongside their member count', async ({mount}) => {
    const component = await mount(<ChannelGovernanceList items={[busy, abandoned]}/>);

    const row = component.locator('[data-testid="governance-row-project-halcyon"]');
    await expect(row).toBeVisible();
    await expect(row).toContainText('61');
});

test('marks channels with no purpose set', async ({mount}) => {
    const component = await mount(<ChannelGovernanceList items={[busy, abandoned]}/>);

    const withPurpose = component.locator('[data-testid="governance-row-platform-standup"]');
    await expect(withPurpose).not.toContainText('Not set');

    const without = component.locator('[data-testid="governance-row-project-halcyon"]');
    await expect(without).toContainText('Not set');
});

test('renders the team-wide labelling coverage summary', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
        />,
    );

    // Denominator is the team, not the single row on screen.
    await expect(component).toContainText('38');
    await expect(component).toContainText('412');
    await expect(component).toContainText('168');
});

test('distinguishes private channels from public', async ({mount}) => {
    const component = await mount(<ChannelGovernanceList items={[busy, priv]}/>);

    await expect(component.locator('[data-testid="governance-row-platform-leads"]')).toContainText('Private');
    await expect(component.locator('[data-testid="governance-row-platform-standup"]')).toContainText('Public');
});

test('renders never-posted channels without a bogus date', async ({mount}) => {
    const component = await mount(<ChannelGovernanceList items={[abandoned]}/>);

    const row = component.locator('[data-testid="governance-row-project-halcyon"]');
    await expect(row).toContainText('Never');
    await expect(row).not.toContainText('1970');
});

test('renders a loading state', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[]}
            loading={true}
        />,
    );
    await expect(component.locator('[data-testid="governance-loading"]')).toBeVisible();
});

test('renders an error state', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[]}
            error='Request failed'
        />,
    );
    await expect(component).toContainText('Request failed');
});

test('renders an empty state', async ({mount}) => {
    const component = await mount(<ChannelGovernanceList items={[]}/>);
    await expect(component.locator('[data-testid="governance-empty"]')).toBeVisible();
});

test('clicking a row calls onSelectChannel with that channel', async ({mount}) => {
    let picked: string | undefined;
    const component = await mount(
        <ChannelGovernanceList
            items={[busy, abandoned]}
            onSelectChannel={(c) => {
                picked = c.name;
            }}
        />,
    );

    await component.locator('[data-testid="governance-row-project-halcyon"]').click();
    expect(picked).toBe('project-halcyon');
});
