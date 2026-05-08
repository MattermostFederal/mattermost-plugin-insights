// Navigation helpers that match what the deprecated insights cards did
// (`<Link to=...>` + a few RHS / popover side effects).
//
// We bridge to the host webapp via the globals exposed by
// `webapp/channels/src/plugins/export.ts` (commit 26617fcbdc):
//   - `window.WebappUtils.browserHistory.push(path)` — same router used
//     by `<Link>`.
//   - `window.ProductApi.selectRhsPost(postId)` — Redux thunk that opens
//     the thread RHS (must be `dispatch`-ed).
//
// Falls back to `window.location.href` / a `<teamName>/pl/<postId>`
// permalink URL when those globals are absent.

interface BrowserHistory {
    push?: (path: string) => void;
}

interface WebappUtils {
    browserHistory?: BrowserHistory;
}

// `selectRhsPost` is a Redux thunk creator: calling it returns a thunk
// that must be passed to `dispatch` for the RHS to actually open.
type Thunk = (dispatch: (action: unknown) => unknown, getState: () => unknown) => unknown;

interface ProductApi {
    selectRhsPost?: (postId: string) => Thunk;
}

interface HostWindow extends Window {
    WebappUtils?: WebappUtils;
    ProductApi?: ProductApi;
}

function host(): HostWindow {
    return window as unknown as HostWindow;
}

export function navigateTo(path: string) {
    const push = host().WebappUtils?.browserHistory?.push;
    if (push) {
        push(path);
        return;
    }
    if (typeof window !== 'undefined') {
        window.location.href = path;
    }
}

// openThreadRHS opens the thread sidebar for the given postId via the
// host's ProductApi. The deprecated insights code dispatched
// `selectPostAndParentChannel(post)` directly; we use the equivalent
// thunk the host exposes (`selectRhsPost`, which wraps `selectPostById`)
// and require the caller's `dispatch` since the helper itself is a thunk.
//
// Returns true when it dispatched an action; the caller can fall back to
// a permalink URL navigation if false.
export function openThreadRHS(
    postId: string,
    dispatch: (action: unknown) => unknown,
): boolean {
    const select = host().ProductApi?.selectRhsPost;
    if (!select) {
        return false;
    }
    dispatch(select(postId));
    return true;
}
