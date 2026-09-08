// Package cache holds the daily snapshot that team insights are served from.
//
// The design constraint is in the requirement itself — team insights are
// "cached server wide once a day". Two consequences shape this package:
//
//   - Keys never include a user. One entry per (insight, team, time range)
//     is shared by everyone on the team, which is only sound because the
//     aggregates cached here are unfiltered and callers apply per-user
//     visibility at read time (see store.PrivateChannelIDsForUser).
//   - Misses are collapsed. Without that, the first page load after a
//     restart lets every concurrent request run the same expensive
//     aggregation at once — the thundering herd the cache exists to prevent.
//
// Entries are held in memory, so each node in a cluster keeps its own copy and
// a restart empties it. That is deliberate: it needs no table, no schema
// migration, and no scheduled job, and the cost is that the first request per
// key after a restart pays for the rebuild while others wait on it.
package cache

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// DefaultTTL is one day, matching the "once a day" requirement.
const DefaultTTL = 24 * time.Hour

// DefaultBuildTimeout bounds a single rebuild. Without it a slow aggregation
// holds a database connection until the client gives up, and every waiter
// behind the singleflight call is blocked for just as long.
const DefaultBuildTimeout = 30 * time.Second

type entry struct {
	value   any
	builtAt time.Time
}

// Cache is a TTL cache with collapsed misses. The zero value is not usable;
// call New.
type Cache struct {
	ttl          time.Duration
	buildTimeout time.Duration

	// now is injected so tests can advance time without sleeping.
	now func() time.Time

	mu    sync.RWMutex
	items map[string]entry
	group singleflight.Group
}

// Options configures a Cache. Zero fields take their defaults.
type Options struct {
	TTL          time.Duration
	BuildTimeout time.Duration
	Now          func() time.Time
}

// New returns a Cache. Passing a zero Options gives a day-long TTL and the
// default build timeout on the real clock.
func New(opts Options) *Cache {
	c := &Cache{
		ttl:          opts.TTL,
		buildTimeout: opts.BuildTimeout,
		now:          opts.Now,
		items:        make(map[string]entry),
	}
	if c.ttl <= 0 {
		c.ttl = DefaultTTL
	}
	if c.buildTimeout <= 0 {
		c.buildTimeout = DefaultBuildTimeout
	}
	if c.now == nil {
		c.now = time.Now
	}
	return c
}

// Key builds a cache key. It takes no user identifier by design: an entry is
// shared across the team, and per-user filtering happens after the read.
func Key(insight, teamID, timeRange string) string {
	return insight + ":" + teamID + ":" + timeRange
}

// Invalidate drops a single entry. Nothing in the request path calls this
// today; it exists for tests and for an eventual admin-triggered refresh.
func (c *Cache) Invalidate(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// Len reports how many entries are held, including any that have expired but
// not yet been replaced. Test and diagnostic use.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *Cache) load(key string) (any, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if c.now().Sub(e.builtAt) >= c.ttl {
		return nil, false
	}
	return e.value, true
}

func (c *Cache) store(key string, value any) {
	c.mu.Lock()
	c.items[key] = entry{value: value, builtAt: c.now()}
	c.mu.Unlock()
}

// GetOrBuild returns the cached value for key, calling build to populate it on
// a miss or once the entry has aged past the TTL.
//
// Concurrent misses on the same key run build once and share the result. The
// build call gets its own timeout-bounded context rather than the caller's:
// the value outlives the request that happened to trigger it, so a client
// disconnecting mid-rebuild must not cancel the work every other waiter is
// blocked on.
//
// A failed build is not cached; the next caller retries.
func GetOrBuild[T any](ctx context.Context, c *Cache, key string, build func(context.Context) (T, error)) (T, error) {
	var zero T

	if v, ok := c.load(key); ok {
		typed, ok := v.(T)
		if !ok {
			// Same key used for two different types — a programming error.
			// Rebuild rather than serve the wrong shape.
			c.Invalidate(key)
		} else {
			return typed, nil
		}
	}

	v, err, _ := c.group.Do(key, func() (any, error) {
		// Re-check: a build that completed while this call waited on the
		// singleflight lock should not be immediately repeated.
		if v, ok := c.load(key); ok {
			if typed, ok := v.(T); ok {
				return typed, nil
			}
		}

		buildCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.buildTimeout)
		defer cancel()

		built, err := build(buildCtx)
		if err != nil {
			return nil, err
		}
		c.store(key, built)
		return built, nil
	})
	if err != nil {
		return zero, err
	}

	typed, ok := v.(T)
	if !ok {
		return zero, nil
	}
	return typed, nil
}
