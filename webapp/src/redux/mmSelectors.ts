// Selectors against the host Mattermost webapp's Redux state. We do not
// import @mattermost/types here; we type the slice we touch and let the
// runtime supply the data.

// Theme is intentionally a partial — themes can be missing fields, and we
// only touch what the user has explicitly set. Field names match
// mattermost-redux's Theme type so the JSON-stringified preference (which
// uses these exact keys) deserializes correctly.
export interface Theme {
    awayIndicator?: string;
    buttonBg?: string;
    buttonColor?: string;
    centerChannelBg?: string;
    centerChannelColor?: string;
    dndIndicator?: string;
    errorTextColor?: string;
    linkColor?: string;
    mentionBg?: string;
    mentionColor?: string;
    mentionHighlightBg?: string;
    mentionHighlightLink?: string;
    newMessageSeparator?: string;
    onlineIndicator?: string;
    sidebarBg?: string;
    sidebarHeaderBg?: string;
    sidebarHeaderTextColor?: string;
    sidebarTeamBarBg?: string;
    sidebarText?: string;
    sidebarTextActiveBorder?: string;
    sidebarTextActiveColor?: string;
    sidebarTextHoverBg?: string;
    sidebarUnreadText?: string;
}

interface PreferenceEntry {
    value?: string;
}

interface TeamRecord {
    id?: string;
    name?: string;
}

// Slim license + config + cloud-subscription shapes — only the fields
// `useLicenseChecks` reads from. Mirrors mattermost-redux's
// selectors/entities/general (`getLicense`, `getConfig`) +
// selectors/entities/cloud (`getCloudSubscription`,
// `getSubscriptionProduct`).
export interface MMLicense {
    IsLicensed?: string;       // "true" | "false"
    IsTrial?: string;          // "true" | "false"
    Cloud?: string;            // "true" | "false"
    Compliance?: string;       // "true" | "false"
    SkuShortName?: string;
}

export interface MMConfig {
    BuildEnterpriseReady?: string;       // "true" | "false"
    EnableComplianceExport?: string;     // "true" | "false"
}

export interface CloudSubscription {
    is_free_trial?: string; // "true" | "false"
}

export interface CloudProduct {
    sku?: string; // 'cloud-starter' | 'cloud-professional' | ...
}

interface MattermostState {
    entities?: {
        teams?: {
            currentTeamId?: string;
            teams?: Record<string, TeamRecord>;
        };
        preferences?: {
            myPreferences?: Record<string, PreferenceEntry>;
        };
        general?: {
            license?: MMLicense;
            config?: MMConfig;
        };
        cloud?: {
            subscription?: CloudSubscription;
            products?: Record<string, CloudProduct>;
        };
        users?: {
            currentUserId?: string;
            profiles?: Record<string, {roles?: string}>;
        };
        channels?: {
            myMembers?: Record<string, unknown>;
        };
    };
}

export function getCurrentUserId(state: unknown): string {
    const root = state as MattermostState;
    return root.entities?.users?.currentUserId ?? '';
}

export function isMemberOfChannel(state: unknown, channelID: string): boolean {
    const root = state as MattermostState;
    return Boolean(root.entities?.channels?.myMembers?.[channelID]);
}

export function getCurrentTeamId(state: unknown): string {
    const root = state as MattermostState;
    return root.entities?.teams?.currentTeamId ?? '';
}

export function getCurrentTeamName(state: unknown): string {
    const root = state as MattermostState;
    const id = root.entities?.teams?.currentTeamId ?? '';
    const teams = root.entities?.teams?.teams ?? {};
    return teams[id]?.name ?? '';
}

export function getLicense(state: unknown): MMLicense {
    const root = state as MattermostState;
    return root.entities?.general?.license ?? {};
}

export function getConfig(state: unknown): MMConfig {
    const root = state as MattermostState;
    return root.entities?.general?.config ?? {};
}

export function getCloudSubscription(state: unknown): CloudSubscription | undefined {
    const root = state as MattermostState;
    return root.entities?.cloud?.subscription;
}

// getCurrentSubscriptionProduct mirrors mattermost-redux's
// `getSubscriptionProduct` selector: it looks up the product the cloud
// subscription's `product_id` points at. Mattermost's redux stores the
// products as a `Record<productID, CloudProduct>` and the active product
// id as the subscription's `product_id`.
export function getCurrentSubscriptionProduct(state: unknown): CloudProduct | undefined {
    const root = state as MattermostState;
    const products = root.entities?.cloud?.products ?? {};
    const sub = root.entities?.cloud?.subscription as (CloudSubscription & {product_id?: string}) | undefined;
    if (!sub?.product_id) {
        return undefined;
    }
    return products[sub.product_id];
}

// getTheme reads the user's current theme from Mattermost's Redux state.
//
// Mattermost stores themes as JSON-encoded preferences keyed by
// `theme--<teamId>` (per-team theme override) or `theme--` (global theme).
// This is a slim port of mattermost-redux/selectors/entities/preferences's
// getTheme — we read the same preference keys but skip pulling in the full
// dependency just to access them.
export function getTheme(state: unknown): Theme | undefined {
    const root = state as MattermostState;
    const prefs = root.entities?.preferences?.myPreferences ?? {};
    const teamId = getCurrentTeamId(state);

    const json = prefs[`theme--${teamId}`]?.value ?? prefs['theme--']?.value;
    if (!json) {
        return undefined;
    }
    try {
        return JSON.parse(json) as Theme;
    } catch {
        return undefined;
    }
}
