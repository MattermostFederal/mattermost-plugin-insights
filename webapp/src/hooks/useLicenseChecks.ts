// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/hooks.ts
// (commit 26617fcbdc).
//
// Adaptations:
//   - `getLicense`, `getConfig`, `getCloudSubscription`,
//     `getSubscriptionProduct` selectors live in `mmSelectors.ts` here
//     (slim local ports of mattermost-redux's selectors).
//   - `isCloudLicense` from `utils/license_utils` → inline check on
//     `license.Cloud === 'true'`. The deprecated helper read the same
//     field with light edge-case handling.
//   - `CloudProducts.STARTER` (a host enum) → the literal string
//     'cloud-starter' the host uses (per mattermost-redux/types/cloud).

import {useSelector} from 'react-redux';

import {getCloudSubscription, getConfig, getCurrentSubscriptionProduct, getLicense} from '../redux/mmSelectors';

const CLOUD_STARTER_SKU = 'cloud-starter';

export interface LicenseChecks {
    isStarterFree: boolean;
    isFreeTrial: boolean;
    isEnterpriseReady: boolean;
}

export function useLicenseChecks(): LicenseChecks {
    const subscription = useSelector(getCloudSubscription);
    const license = useSelector(getLicense);
    const subscriptionProduct = useSelector(getCurrentSubscriptionProduct);
    const config = useSelector(getConfig);

    const isCloud = license.Cloud === 'true';
    const isCloudStarterFree = isCloud && subscriptionProduct?.sku === CLOUD_STARTER_SKU;
    const isCloudFreeTrial = isCloud && subscription?.is_free_trial === 'true';

    const isEnterpriseReady = config.BuildEnterpriseReady === 'true';
    const isSelfHostedStarter = isEnterpriseReady && license.IsLicensed === 'false';
    const isSelfHostedFreeTrial = license.IsTrial === 'true';

    return {
        isStarterFree: isCloudStarterFree || isSelfHostedStarter,
        isFreeTrial: isCloudFreeTrial || isSelfHostedFreeTrial,
        isEnterpriseReady,
    };
}

// Mirrors the deprecated `useGetFilterType` (insights/hooks.ts). When the
// server is starter-free or non-enterprise, the user can never view team
// insights, so we lock the scope to MY regardless of what the caller
// passed in. The deprecated hook also persisted the chosen scope in
// global state; we keep that responsibility on the caller.
export function useEffectiveScope(scope: 'my' | 'team'): 'my' | 'team' {
    const {isStarterFree, isEnterpriseReady} = useLicenseChecks();
    if (isStarterFree || !isEnterpriseReady) {
        return 'my';
    }
    return scope;
}
