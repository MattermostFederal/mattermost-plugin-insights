import {expect, test} from '@playwright/experimental-ct-react';
import React from 'react';

import {ScopeSelect} from './ScopeSelect';

test('renders the current scope label', async ({mount}) => {
    const component = await mount(
        <ScopeSelect
            value='my'
            onChange={() => undefined}
        />,
    );
    await expect(component.locator('.insights-scope__title')).toContainText('My Insights');
});

test('clicking the title opens the menu and selecting an item triggers onChange', async ({mount}) => {
    let chosen: string | undefined;
    const component = await mount(
        <ScopeSelect
            value='my'
            onChange={(s) => {
                chosen = s;
            }}
        />,
    );
    await expect(component.locator('.insights-scope__menu')).toHaveCount(0);
    await component.locator('.insights-scope__title').click();
    await expect(component.locator('.insights-scope__menu')).toHaveCount(1);
    await component.locator('.insights-scope__menu-item', {hasText: 'Team Insights'}).click();
    expect(chosen).toBe('team');
    await expect(component.locator('.insights-scope__menu')).toHaveCount(0);
});

test('marks the current scope as selected in the menu', async ({mount}) => {
    const component = await mount(
        <ScopeSelect
            value='team'
            onChange={() => undefined}
        />,
    );
    await component.locator('.insights-scope__title').click();
    await expect(component.locator('.insights-scope__menu-item.is-selected')).toContainText('Team Insights');
});

function stateWithLicense(license: Record<string, string>) {
    return {
        entities: {
            general: {
                license,
                config: {BuildEnterpriseReady: 'true'},
            },
            users: {
                currentUserId: 'u1',
                profiles: {u1: {roles: 'system_user'}},
            },
            cloud: {},
        },
    };
}

test('shows a lock badge on the Team Insights menu item when license is starter-free', async ({mount}) => {
    const component = await mount(
        <ScopeSelect
            value='my'
            onChange={() => undefined}
        />,
        {hooksConfig: {state: stateWithLicense({IsLicensed: 'false'})}},
    );
    await component.locator('.insights-scope__title').click();
    await expect(component.locator('.insights-scope__menu-item.is-restricted')).toHaveCount(1);
    await expect(component.locator('.insights-scope__menu-item.is-restricted i.icon-lock-outline')).toHaveCount(1);
});

test('clicking the locked Team Insights item opens the access modal and does NOT call onChange', async ({mount, page}) => {
    let chosen: string | undefined;
    const component = await mount(
        <ScopeSelect
            value='my'
            onChange={(s) => {
                chosen = s;
            }}
        />,
        {hooksConfig: {state: stateWithLicense({IsLicensed: 'false'})}},
    );
    await component.locator('.insights-scope__title').click();
    // The locked item is aria-disabled (Playwright auto-waits for
    // enabled), but the deprecated UX intentionally leaves it clickable
    // so the access modal can open. Force the click.
    await component.locator('.insights-scope__menu-item.is-restricted').click({force: true});
    expect(chosen).toBeUndefined();

    // The bootstrap modal portals to body, so the modal title shows up at
    // page level, not inside `component`.
    await expect(page.locator('.insights-access-modal')).toBeVisible();
});

test('on a Professional license the Team Insights item is not restricted', async ({mount}) => {
    let chosen: string | undefined;
    const component = await mount(
        <ScopeSelect
            value='my'
            onChange={(s) => {
                chosen = s;
            }}
        />,
        {hooksConfig: {state: stateWithLicense({IsLicensed: 'true', SkuShortName: 'professional'})}},
    );
    await component.locator('.insights-scope__title').click();
    await expect(component.locator('.insights-scope__menu-item.is-restricted')).toHaveCount(0);
    await component.locator('.insights-scope__menu-item', {hasText: 'Team Insights'}).click();
    expect(chosen).toBe('team');
});
