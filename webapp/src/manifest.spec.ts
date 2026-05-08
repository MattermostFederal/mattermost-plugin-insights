import {expect, test} from '@playwright/test';
import manifest from 'manifest';

test('manifest has the expected plugin id', () => {
    expect(manifest.id).toBe('insights');
});
