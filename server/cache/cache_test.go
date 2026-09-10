package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock lets the tests age entries past the TTL without sleeping.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func TestGetOrBuild_buildsOnceThenServesFromCache(t *testing.T) {
	clock := newClock()
	c := New(Options{TTL: time.Hour, Now: clock.Now})

	var builds atomic.Int32
	build := func(context.Context) (string, error) {
		builds.Add(1)
		return "value", nil
	}

	for i := range 5 {
		got, err := GetOrBuild(context.Background(), c, "k", build)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if got != "value" {
			t.Fatalf("call %d: got %q; want %q", i, got, "value")
		}
	}
	if n := builds.Load(); n != 1 {
		t.Errorf("build ran %d times; want 1", n)
	}
}

func TestGetOrBuild_rebuildsAfterTTL(t *testing.T) {
	clock := newClock()
	c := New(Options{TTL: 24 * time.Hour, Now: clock.Now})

	var builds atomic.Int32
	build := func(context.Context) (int, error) {
		return int(builds.Add(1)), nil
	}

	first, _ := GetOrBuild(context.Background(), c, "k", build)

	// Just short of the TTL the original entry still stands.
	clock.Advance(24*time.Hour - time.Second)
	second, _ := GetOrBuild(context.Background(), c, "k", build)
	if second != first {
		t.Errorf("value changed before TTL elapsed: %d then %d", first, second)
	}

	clock.Advance(2 * time.Second)
	third, _ := GetOrBuild(context.Background(), c, "k", build)
	if third == first {
		t.Errorf("value did not rebuild after TTL: still %d", third)
	}
	if n := builds.Load(); n != 2 {
		t.Errorf("build ran %d times; want 2", n)
	}
}

// The thundering-herd case: a cold cache hit by many requests at once must run
// the aggregation once, not once per request.
func TestGetOrBuild_collapsesConcurrentMisses(t *testing.T) {
	c := New(Options{TTL: time.Hour, Now: newClock().Now})

	var builds atomic.Int32
	release := make(chan struct{})
	build := func(context.Context) (string, error) {
		builds.Add(1)
		<-release // hold every caller inside the build
		return "value", nil
	}

	const callers = 50
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for range callers {
		wg.Go(func() {
			got, err := GetOrBuild(context.Background(), c, "k", build)
			if err != nil {
				errs <- err
				return
			}
			if got != "value" {
				errs <- errors.New("wrong value: " + got)
			}
		})
	}

	// Let the goroutines pile up on the same key before the build returns.
	for c.Len() == 0 && builds.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	close(release)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("caller: %v", err)
	}
	if n := builds.Load(); n != 1 {
		t.Errorf("build ran %d times under %d concurrent callers; want 1", n, callers)
	}
}

func TestGetOrBuild_doesNotCacheFailures(t *testing.T) {
	c := New(Options{TTL: time.Hour, Now: newClock().Now})

	var builds atomic.Int32
	boom := errors.New("boom")
	build := func(context.Context) (string, error) {
		if builds.Add(1) == 1 {
			return "", boom
		}
		return "recovered", nil
	}

	if _, err := GetOrBuild(context.Background(), c, "k", build); !errors.Is(err, boom) {
		t.Fatalf("first call err = %v; want boom", err)
	}
	if c.Len() != 0 {
		t.Error("a failed build was cached")
	}

	got, err := GetOrBuild(context.Background(), c, "k", build)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got != "recovered" {
		t.Errorf("got %q; want %q", got, "recovered")
	}
}

// A client that disconnects mid-rebuild returns promptly, while the detached
// build continues and populates the cache for the next caller.
func TestGetOrBuild_canceledWaiterReturnsWhileBuildContinues(t *testing.T) {
	c := New(Options{TTL: time.Hour, Now: newClock().Now})

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	release := make(chan struct{})
	built := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		_, err := GetOrBuild(ctx, c, "k", func(buildCtx context.Context) (string, error) {
			close(started)
			<-release
			defer close(built)
			if err := buildCtx.Err(); err != nil {
				return "", err
			}
			return "built", nil
		})
		result <- err
	}()

	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled waiter err = %v; want context.Canceled", err)
	}

	close(release)
	<-built
	got, err := GetOrBuild(context.Background(), c, "k", func(buildCtx context.Context) (string, error) {
		if err := buildCtx.Err(); err != nil {
			return "", err
		}
		return "rebuilt", nil
	})
	if err != nil {
		t.Fatalf("cached read: %v", err)
	}
	if got != "built" {
		t.Errorf("got %q; want the completed detached build", got)
	}
}

func TestGetOrBuild_buildTimeoutApplies(t *testing.T) {
	c := New(Options{TTL: time.Hour, BuildTimeout: 20 * time.Millisecond, Now: newClock().Now})

	_, err := GetOrBuild(context.Background(), c, "k", func(buildCtx context.Context) (string, error) {
		<-buildCtx.Done()
		return "", buildCtx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v; want DeadlineExceeded", err)
	}
}

func TestGetOrBuild_keysAreIndependent(t *testing.T) {
	c := New(Options{TTL: time.Hour, Now: newClock().Now})

	a, _ := GetOrBuild(context.Background(), c, Key("channels", "team1", "7_day"), func(context.Context) (string, error) {
		return "a", nil
	})
	b, _ := GetOrBuild(context.Background(), c, Key("channels", "team1", "28_day"), func(context.Context) (string, error) {
		return "b", nil
	})
	if a != "a" || b != "b" {
		t.Errorf("got %q/%q; want a/b", a, b)
	}
	if c.Len() != 2 {
		t.Errorf("Len = %d; want 2", c.Len())
	}
}

// The key must not vary by user: one entry serves the whole team, and
// per-user visibility is applied after the read.
func TestKey_excludesUser(t *testing.T) {
	if got, want := Key("channels", "team1", "7_day"), "channels:team1:7_day"; got != want {
		t.Errorf("Key = %q; want %q", got, want)
	}
}

func TestNew_appliesDefaults(t *testing.T) {
	c := New(Options{})
	if c.ttl != DefaultTTL {
		t.Errorf("ttl = %v; want %v", c.ttl, DefaultTTL)
	}
	if c.buildTimeout != DefaultBuildTimeout {
		t.Errorf("buildTimeout = %v; want %v", c.buildTimeout, DefaultBuildTimeout)
	}
	if c.now == nil {
		t.Error("now is nil")
	}
}

// An expired entry that is never read again must not stay resident. Without
// a sweep the map grows with every team whose page is opened once.
func TestGetOrBuild_expiredEntriesAreEvictedOnWrite(t *testing.T) {
	clock := newClock()
	c := New(Options{TTL: time.Hour, Now: clock.Now})

	build := func(context.Context) (string, error) { return "v", nil }

	// Populate several keys that will never be requested again.
	for _, k := range []string{"team1", "team2", "team3"} {
		if _, err := GetOrBuild(context.Background(), c, k, build); err != nil {
			t.Fatalf("populate %s: %v", k, err)
		}
	}
	if c.Len() != 3 {
		t.Fatalf("Len = %d; want 3", c.Len())
	}

	// Age them all out, then write one unrelated key.
	clock.Advance(2 * time.Hour)
	if _, err := GetOrBuild(context.Background(), c, "team4", build); err != nil {
		t.Fatalf("populate team4: %v", err)
	}

	if got := c.Len(); got != 1 {
		t.Errorf("Len = %d after the stale keys aged out; want 1", got)
	}
}

func TestInvalidate(t *testing.T) {
	c := New(Options{TTL: time.Hour, Now: newClock().Now})
	build := func(context.Context) (string, error) { return "v", nil }

	if _, err := GetOrBuild(context.Background(), c, "k", build); err != nil {
		t.Fatalf("populate: %v", err)
	}
	if c.Len() != 1 {
		t.Fatalf("Len = %d; want 1", c.Len())
	}
	c.Invalidate("k")
	if c.Len() != 0 {
		t.Errorf("Len = %d after Invalidate; want 0", c.Len())
	}
}
