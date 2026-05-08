import systemEmojis from './system_emojis.json';

const map = systemEmojis as Record<string, string>;

// Returns the public image URL Mattermost serves for this emoji. Mirrors
// `Client4.getSystemEmojiImageUrl` / `getEmojiImageUrl` from the host
// webapp's emoji_utils:
//
//   - system emoji: `/static/emoji/<unified-codepoints>.png`
//   - custom emoji: `/api/v4/emoji/name/<name>/image` (server resolves)
//
// Returns null when we have no idea what the name maps to. The renderer
// can then render an `:emoji_name:` text fallback.
export function getEmojiImageUrl(name: string): string | null {
    const unified = map[name];
    if (unified) {
        return `/static/emoji/${unified}.png`;
    }

    // Custom emoji: rely on the server-side name lookup.
    return `/api/v4/emoji/name/${encodeURIComponent(name)}/image`;
}

// preloadEmojis warms the browser's HTTP cache for the given emoji names by
// fetching the image URL for each unknown name in the system map. This
// mirrors what the deprecated `loadCustomEmojisIfNeeded(names)` dispatch
// did on the host: populate a runtime cache so subsequent renders are
// instant. Without it, every <img> mount would block on a fresh fetch.
//
// Names already in the bundled system-emoji map are skipped — those
// resolve to a static URL the browser tends to have cached anyway.
//
// Fire-and-forget; failures are swallowed.
export function preloadEmojis(names: string[]): void {
    if (typeof window === 'undefined' || !names.length) {
        return;
    }
    const seen = new Set<string>();
    for (const name of names) {
        if (seen.has(name) || map[name]) {
            continue;
        }
        seen.add(name);
        const url = `/api/v4/emoji/name/${encodeURIComponent(name)}/image`;
        // Image() preload is intentional — it triggers the same browser
        // cache fill as the eventual <img src=...> render.
        const img = new Image();
        img.src = url;
    }
}
