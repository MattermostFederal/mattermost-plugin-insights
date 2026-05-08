import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {TopThreadsList} from './TopThreadsList';

const sampleThread = {
    channel_id: 'ch1',
    channel_display_name: 'Project planning',
    channel_name: 'project-planning',
    participants: ['u1', 'u2', 'u3'],
    user_information: {
        id: 'author1',
        username: 'jane.doe',
        first_name: 'Jane',
        last_name: 'Doe',
        nickname: 'jd',
        last_picture_update: 0,
    },
    post: {id: 'post1', message: 'This is the first message in the thread', reply_count: 12},
};

test('shows empty state when there are no threads', async ({mount}) => {
    const component = await mount(
        <TopThreadsList
            items={[]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component).toContainText('Not enough data');
    await expect(component.locator('i.icon-message-text-outline')).toHaveCount(1);
});

test('shows skeleton when status is loading', async ({mount}) => {
    const component = await mount(
        <TopThreadsList
            items={[]}
            hasNext={false}
            status='loading'
        />,
    );
    await expect(component.locator('.top-thread-loading-container')).toHaveCount(3);
});

test('clicking a thread row dispatches the host RHS thunk', async ({mount, page}) => {
    const component = await mount(
        <TopThreadsList
            items={[sampleThread]}
            hasNext={false}
            status='idle'
            teamName='ad-1'
        />,
    );
    await page.evaluate(() => {
        const w = window as unknown as {ProductApi?: {selectRhsPost?: (id: string) => unknown}; __thunkPostId?: string};
        w.ProductApi = {
            selectRhsPost: (postId: string) => () => {
                w.__thunkPostId = postId;
            },
        };
    });
    await component.locator('.thread-item').first().click();
    const captured = await page.evaluate(() => (window as unknown as {__thunkPostId?: string}).__thunkPostId);
    expect(captured).toBe('post1');
});

test('row reply-count shows post.reply_count, not participants.length', async ({mount}) => {
    // Regression: the deprecated card rendered `thread.post.reply_count`
    // (the actual reply tally), not `thread.participants.length` (a
    // different metric: count of distinct posters).
    const component = await mount(
        <TopThreadsList
            items={[{
                ...sampleThread,
                participants: ['u1', 'u2', 'u3'], // 3 unique posters
                post: {id: 'post1', message: 'm', reply_count: 68}, // 68 actual replies
            }]}
            hasNext={false}
            status='idle'
        />,
    );
    await expect(component.locator('.reply-count')).toContainText('68');
});

test('non-member click opens JoinChannelModal when compliance export is enabled', async ({mount, page}) => {
    const component = await mount(
        <TopThreadsList
            items={[sampleThread]}
            hasNext={false}
            status='idle'
            teamName='ad-1'
        />,
        {hooksConfig: {state: {
            entities: {
                users: {currentUserId: 'me'},
                general: {
                    license: {Compliance: 'true'},
                    config: {EnableComplianceExport: 'true', BuildEnterpriseReady: 'true'},
                },
                channels: {myMembers: {}}, // user is NOT a member of any channel
                cloud: {},
            },
        }}},
    );
    await component.locator('.thread-item').first().click();

    // Modal portals to body — assert at page level.
    await expect(page.getByText('Join channel?')).toBeVisible();
});

test('non-member click on a non-compliance server skips the modal and goes straight to RHS', async ({mount, page}) => {
    const component = await mount(
        <TopThreadsList
            items={[sampleThread]}
            hasNext={false}
            status='idle'
            teamName='ad-1'
        />,
        {hooksConfig: {state: {
            entities: {
                users: {currentUserId: 'me'},
                general: {license: {}, config: {BuildEnterpriseReady: 'true'}},
                channels: {myMembers: {}},
                cloud: {},
            },
        }}},
    );
    await page.evaluate(() => {
        const w = window as unknown as {ProductApi?: {selectRhsPost?: (id: string) => unknown}; __thunkPostId?: string};
        w.ProductApi = {
            selectRhsPost: (postId: string) => () => {
                w.__thunkPostId = postId;
            },
        };
    });
    await component.locator('.thread-item').first().click();
    const captured = await page.evaluate(() => (window as unknown as {__thunkPostId?: string}).__thunkPostId);
    expect(captured).toBe('post1');
    await expect(page.getByText('Join channel?')).toHaveCount(0);
});

test('row renders avatar, author, channel tag, and post-message preview', async ({mount}) => {
    const component = await mount(
        <TopThreadsList
            items={[sampleThread]}
            hasNext={false}
            status='idle'
        />,
    );
    const row = component.locator('.thread-item').first();
    await expect(row).toBeVisible();

    // Avatar
    await expect(row.locator('.Avatar')).toHaveCount(1);

    // Author display name
    await expect(row.locator('.display-name')).toContainText('Jane Doe');

    // Channel pill
    await expect(row.locator('.Tag')).toContainText('Project planning');

    // Post message preview
    await expect(row.locator('.preview')).toContainText('This is the first message in the thread');
});
