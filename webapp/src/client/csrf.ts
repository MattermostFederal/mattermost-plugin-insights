// CSRF handling for the plugin's own fetch calls.
//
// The plugin does not bundle mattermost-redux, so every request is a bare
// `fetch` rather than a host `Client4` call. Client4 attaches a CSRF token
// for us; a bare fetch does not, and the plugin was relying instead on the
// deprecated `X-Requested-With: XMLHttpRequest` escape hatch. The server
// still honours it, but logs on every request and refuses outright when
// `ExperimentalStrictCSRFEnforcement` is on:
//
//   server/channels/app/plugin_requests.go, validateCSRFForPluginRequest:
//     // ToDo(DSchalla) 2019/01/04: Remove after deprecation period and
//     // only allow CSRF Header (MM-13657)
//
// The header set here is the same one the host builds in
// webapp/platform/client/src/client4.ts `getOptions` — legacy header always,
// CSRF token from the MMCSRF cookie on state-changing methods only. GET is
// exempt server-side, so it carries no token.

const CSRF_COOKIE = 'MMCSRF';
const CSRF_HEADER = 'X-CSRF-Token';
const REQUESTED_WITH_HEADER = 'X-Requested-With';

// Takes the raw `document.cookie` string rather than reading it, so the
// parsing is testable without a browser.
export function parseCSRFToken(cookie: string): string {
    for (const entry of cookie.split(';')) {
        const trimmed = entry.trim();
        if (trimmed.startsWith(`${CSRF_COOKIE}=`)) {
            return trimmed.slice(CSRF_COOKIE.length + 1);
        }
    }
    return '';
}

export function getCSRFToken(): string {
    if (typeof document === 'undefined' || typeof document.cookie === 'undefined') {
        return '';
    }
    return parseCSRFToken(document.cookie);
}

// `readToken` is injectable for the same reason: the tests drive it directly
// instead of standing up a document.
export function requestHeaders(
    method: string,
    extra: Record<string, string> = {},
    readToken: () => string = getCSRFToken,
): Record<string, string> {
    const headers: Record<string, string> = {

        // Kept as the fallback for the case the token is missing — a server
        // without strict enforcement still accepts the request on this alone.
        [REQUESTED_WITH_HEADER]: 'XMLHttpRequest',
        ...extra,
    };

    const token = readToken();
    if (method.toLowerCase() !== 'get' && token) {
        headers[CSRF_HEADER] = token;
    }

    return headers;
}
