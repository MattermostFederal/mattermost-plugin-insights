import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {NewTeamMembersList} from './NewTeamMembersList';

const sampleMember = {
    id: 'u1',
    username: 'alice',
    first_name: 'Alice',
    last_name: 'Anders',
    nickname: 'ali',
    position: 'Engineer',
    create_at: 1_700_000_000_000,
};

test('shows skeleton when status is loading', async ({mount}) => {
    const component = await mount(
        <NewTeamMembersList
            items={[]}
            hasNext={false}
            status='loading'
        />,
    );
    await expect(component.locator('.new-members-loading-container')).toHaveCount(5);
});

test('shows empty state when no members in the window', async ({mount}) => {
    const component = await mount(
        <NewTeamMembersList
            items={[]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component).toContainText('Not enough data');
    await expect(component.locator('i.icon-account-multiple-outline')).toHaveCount(1);
});

test('row renders avatar, display name, position, and Say hello CTA', async ({mount}) => {
    const component = await mount(
        <NewTeamMembersList
            items={[sampleMember]}
            hasNext={false}
            status='idle'
        />,
    );
    const row = component.locator('.new-members-item').first();
    await expect(row).toBeVisible();

    // Avatar
    await expect(row.locator('.Avatar')).toHaveCount(1);

    // Display name + position
    await expect(row.locator('.dm-name')).toContainText('Alice Anders');
    await expect(row.locator('.dm-role')).toContainText('Engineer');

    // Say hello CTA
    await expect(row.locator('.say-hello')).toContainText('Say hello');
});

test('falls back to username when no first/last name', async ({mount}) => {
    const component = await mount(
        <NewTeamMembersList
            items={[{...sampleMember, first_name: '', last_name: ''}]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component.locator('.dm-name')).toContainText('alice');
});

test('omits position when empty', async ({mount}) => {
    const component = await mount(
        <NewTeamMembersList
            items={[{...sampleMember, position: ''}]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component.locator('.dm-role')).toHaveCount(0);
});

// Total-count rendering moved to `NewMembersTotal` (see
// NewMembersTotal.pw.tsx) — the list no longer shows it.
