// Telemetry helper. The deprecated webapp called
// `trackEvent('insights', '<event>', properties?)` from
// `actions/telemetry_actions`, which the host's RudderStack integration
// forwarded to the analytics backend. The plugin doesn't have access to
// that pipeline, so this helper POSTs to the plugin's own
// `/api/v1/telemetry` endpoint (see server/api/telemetry.go), where the
// event is recorded to the Mattermost server log for downstream
// analytics scraping.
//
// Failures are swallowed: a missing telemetry event must not break a
// user-visible action like opening a channel or thread.

const url = '/plugins/insights/api/v1/telemetry';

export function trackInsightsEvent(event: string, properties?: Record<string, unknown>): void {
    try {
        // Fire-and-forget — fetch returns a promise but we never await
        // because telemetry isn't in the user-action critical path.
        void fetch(url, {
            method: 'POST',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
                'X-Requested-With': 'XMLHttpRequest',
            },
            body: JSON.stringify({event, properties}),
        }).catch(() => undefined);
    } catch {
        // Same swallow.
    }
}
