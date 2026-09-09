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
    matching_channels: 412,
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

// The tiles report the work remaining, not the work done — "374 missing a
// purpose" is a queue you can act on; "38 labelled" is trivia.
test('summarises the work remaining across the whole team', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
        />,
    );

    const tiles = component.locator('[data-testid="governance-summary"]');
    await expect(tiles).toContainText('412'); // channels in team
    await expect(tiles).toContainText('244'); // 412 - 168 active
    await expect(tiles).toContainText('374'); // 412 - 38 labelled

    // Denominators describe the team, not the single row on screen.
    await expect(tiles).toContainText('9% of 412 labelled');
});

test('renders a search box and quick filters', async ({mount}) => {
    let searched: string | undefined;
    let filtered: string | undefined;
    const component = await mount(
        <ChannelGovernanceList
            items={[busy, abandoned]}
            summary={summary}
            search=''
            onSearch={(v) => {
                searched = v;
            }}
            filter=''
            onFilter={(v) => {
                filtered = v;
            }}
        />,
    );

    await component.locator('[data-testid="governance-search"]').fill('deploy');
    expect(searched).toBe('deploy');

    await component.getByRole('button', {name: 'Missing a purpose'}).click();
    expect(filtered).toBe('unlabelled');
});

// A narrowed view must say so, or the page looks like the team shrank.
test('shows a match count only when the view is narrowed', async ({mount}) => {
    const all = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
            onSearch={() => undefined}
        />,
    );
    await expect(all.locator('[data-testid="governance-matchcount"]')).toHaveCount(0);

    await all.unmount();

    const narrowed = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={{...summary, matching_channels: 7}}
            onSearch={() => undefined}
        />,
    );
    await expect(narrowed.locator('[data-testid="governance-matchcount"]')).toContainText('7 of 412');
});

// The defining property of the product is that numbers are up to a day old.
test('states how old the snapshot is', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
            generatedAt={1757289600000}
        />,
    );
    await expect(component.locator('[data-testid="governance-freshness"]')).toContainText('refreshed daily');
});

test('omits the freshness line when the snapshot was just built', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
            generatedAt={0}
        />,
    );
    await expect(component.locator('[data-testid="governance-freshness"]')).toHaveCount(0);
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

test('column headers are clickable and report the chosen column', async ({mount}) => {
    let picked: string | undefined;
    const component = await mount(
        <ChannelGovernanceList
            items={[busy, abandoned]}
            sort='posts'
            ascending={false}
            onSort={(c) => {
                picked = c;
            }}
        />,
    );

    await component.getByRole('button', {name: /Members/}).click();
    expect(picked).toBe('members');
});

// Sorting is what lets the quiet channels be found at all — without it they
// sit on the last page of a fixed descending list.
test('marks the active sort column for assistive tech', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy, abandoned]}
            sort='members'
            ascending={true}
            onSort={() => undefined}
        />,
    );

    const active = component.locator('th[aria-sort="ascending"]');
    await expect(active).toHaveCount(1);
    await expect(active).toContainText('Members');
});

// Without an onSort handler the headers stay plain text, so the component is
// still usable read-only.
test('renders plain headers when sorting is not wired up', async ({mount}) => {
    const component = await mount(<ChannelGovernanceList items={[busy]}/>);
    await expect(component.locator('th[aria-sort]')).toHaveCount(0);
    await expect(component).toContainText('Members');
});

// New Team Members joins the summary row as a tile rather than sitting below
// the table, where six joiners out-sized eleven channels.
test('renders a trailing tile in the summary row', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
            trailingTile={<div data-testid='extra-tile'>{'7 joined'}</div>}
        />,
    );

    const tiles = component.locator('[data-testid="governance-summary"]');
    await expect(tiles.locator('[data-testid="extra-tile"]')).toContainText('7 joined');
});

test('summary row stays three-up when no trailing tile is given', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
        />,
    );
    await expect(component.locator('.governance-summary--four')).toHaveCount(0);
});

// --- timeRange-aware "no posts" label ---
// The stat tile and inactive filter chip both derive from noPostsLabel(), so
// each TimeRange branch is exercised once through the rendered output rather
// than calling the private function directly.

test('stat tile reads "No posts since yesterday" for the 1-day window', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
            timeRange='1_day'
        />,
    );
    await expect(component.locator('[data-testid="governance-summary"]')).toContainText('No posts since yesterday');
});

test('stat tile reads "No posts in the last 7 days" for the 7-day window', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
            timeRange='7_day'
        />,
    );
    await expect(component.locator('[data-testid="governance-summary"]')).toContainText('No posts in the last 7 days');
});

test('stat tile reads "No posts in the last 28 days" for the 28-day window', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
            timeRange='28_day'
        />,
    );
    await expect(component.locator('[data-testid="governance-summary"]')).toContainText('No posts in the last 28 days');
});

test('stat tile falls back gracefully when timeRange is omitted', async ({mount}) => {
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
        />,
    );
    await expect(component.locator('[data-testid="governance-summary"]')).toContainText('No posts in selected window');
});

// The filter chip label must match the stat tile so the two surfaces agree.

test('inactive filter chip label matches the stat tile for each window', async ({mount}) => {
    for (const [timeRange, expected] of [
        ['1_day', 'No posts since yesterday'],
        ['7_day', 'No posts in the last 7 days'],
        ['28_day', 'No posts in the last 28 days'],
        [undefined, 'No posts in selected window'],
    ] as const) {
        const component = await mount(
            <ChannelGovernanceList
                items={[busy]}
                summary={summary}
                filter=''
                onFilter={() => undefined}
                timeRange={timeRange}
            />,
        );
        await expect(component.getByRole('button', {name: expected})).toBeVisible();
        await component.unmount();
    }
});

test('clicking the inactive filter chip fires onFilter with "inactive"', async ({mount}) => {
    let filtered: string | undefined;
    const component = await mount(
        <ChannelGovernanceList
            items={[busy]}
            summary={summary}
            filter=''
            onFilter={(v) => {
                filtered = v;
            }}
            timeRange='1_day'
        />,
    );
    await component.getByRole('button', {name: 'No posts since yesterday'}).click();
    expect(filtered).toBe('inactive');
});
