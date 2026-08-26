package throttle

import (
	"fmt"
	"sync"
	"testing"
	"time"
	"web-app/app/services/throttle"
)

/*
 * These need no server and no database, and never sleep: Allow takes the current
 * time as a parameter, so elapsed time is expressed by passing a later one. A
 * rate limiter tested by sleeping is a slow test that also cannot assert on exact
 * refill arithmetic.
 */

// base is a fixed instant every case advances from, so no test depends on the
// wall clock.
var base = time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)

const testKey = "global|203.0.113.7"

func perMinute(requests int) throttle.Limit {
	return throttle.Limit{Name: "global", Requests: requests, Per: time.Minute}
}

// The allowance is exactly the configured burst, and the request after it is
// refused.
func TestBurstIsExactlyTheLimit(t *testing.T) {
	store := throttle.NewMemoryStore(0)
	limit := perMinute(5)

	for attempt := 1; attempt <= limit.Requests; attempt++ {
		decision := store.Allow(testKey, limit, base)
		if !decision.Allowed {
			t.Fatalf("attempt %d was refused inside the allowance of %d", attempt, limit.Requests)
		}

		if want := limit.Requests - attempt; decision.Remaining != want {
			t.Errorf("attempt %d: remaining = %d, want %d", attempt, decision.Remaining, want)
		}
	}

	decision := store.Allow(testKey, limit, base)
	if decision.Allowed {
		t.Fatalf("attempt %d was allowed beyond the allowance of %d", limit.Requests+1, limit.Requests)
	}

	if decision.Remaining != 0 {
		t.Errorf("remaining = %d on a refusal, want 0", decision.Remaining)
	}
}

/*
 * Tokens return with elapsed time rather than all at once on a window boundary.
 * At sixty per minute one token is worth one second, so a single second of
 * waiting buys exactly one request and no more.
 */
func TestTokensRefillWithElapsedTime(t *testing.T) {
	store := throttle.NewMemoryStore(0)
	limit := perMinute(60)

	for range limit.Requests {
		if !store.Allow(testKey, limit, base).Allowed {
			t.Fatal("a request inside the allowance was refused")
		}
	}

	if store.Allow(testKey, limit, base).Allowed {
		t.Fatal("the allowance was not exhausted")
	}

	oneSecondLater := base.Add(time.Second)

	if !store.Allow(testKey, limit, oneSecondLater).Allowed {
		t.Fatal("a second of waiting did not return a token")
	}

	if store.Allow(testKey, limit, oneSecondLater).Allowed {
		t.Fatal("a second of waiting returned more than one token")
	}
}

// RetryAfter has to be the real wait. A value that is too short sends a client
// back to be refused again; too long and it waits needlessly.
func TestRetryAfterIsTheWaitForOneToken(t *testing.T) {
	store := throttle.NewMemoryStore(0)
	limit := perMinute(60) // one token per second

	for range limit.Requests {
		store.Allow(testKey, limit, base)
	}

	decision := store.Allow(testKey, limit, base)
	if decision.Allowed {
		t.Fatal("the allowance was not exhausted")
	}

	if decision.RetryAfter != time.Second {
		t.Fatalf("retry after = %s, want %s", decision.RetryAfter, time.Second)
	}

	// Half a token in: the remaining wait is the other half.
	decision = store.Allow(testKey, limit, base.Add(500*time.Millisecond))
	if decision.Allowed {
		t.Fatal("half a token was enough to be allowed")
	}

	if decision.RetryAfter != 500*time.Millisecond {
		t.Fatalf("retry after = %s, want %s", decision.RetryAfter, 500*time.Millisecond)
	}
}

// ResetAt is when the whole allowance is back, not when the next token lands.
func TestResetAtIsFullReplenishment(t *testing.T) {
	store := throttle.NewMemoryStore(0)
	limit := perMinute(60)

	decision := store.Allow(testKey, limit, base)

	// One token spent of sixty, refilling at one per second.
	if want := base.Add(time.Second); !decision.ResetAt.Equal(want) {
		t.Fatalf("reset at = %s, want %s", decision.ResetAt, want)
	}

	for range limit.Requests {
		decision = store.Allow(testKey, limit, base)
	}

	if want := base.Add(limit.Per); !decision.ResetAt.Equal(want) {
		t.Fatalf("reset at = %s on an exhausted allowance, want %s", decision.ResetAt, want)
	}
}

/*
 * The property that makes per-route limits work at all. Two limits applied to
 * one caller must count separately, or a tight limit and a loose one would draw
 * down the same tokens and both would be stricter than configured.
 */
func TestLimitsWithDifferentNamesDoNotShareAnAllowance(t *testing.T) {
	store := throttle.NewMemoryStore(0)

	const address = "203.0.113.7"

	login := throttle.Limit{Name: "login", Requests: 2, Per: time.Minute}
	global := throttle.Limit{Name: "global", Requests: 10, Per: time.Minute}

	for range login.Requests {
		if !store.Allow(login.Key(address), login, base).Allowed {
			t.Fatal("a request inside the login allowance was refused")
		}
	}

	if store.Allow(login.Key(address), login, base).Allowed {
		t.Fatal("the login allowance was not exhausted")
	}

	// The global allowance saw none of that.
	decision := store.Allow(global.Key(address), global, base)
	if !decision.Allowed {
		t.Fatal("exhausting the login allowance also exhausted the global one")
	}

	if want := global.Requests - 1; decision.Remaining != want {
		t.Fatalf("global remaining = %d, want %d; the login limit consumed from it", decision.Remaining, want)
	}
}

// One caller running out must not affect another.
func TestCallersAreCountedIndependently(t *testing.T) {
	store := throttle.NewMemoryStore(0)
	limit := perMinute(2)

	for range limit.Requests {
		store.Allow(limit.Key("203.0.113.7"), limit, base)
	}

	if store.Allow(limit.Key("203.0.113.7"), limit, base).Allowed {
		t.Fatal("the first caller's allowance was not exhausted")
	}

	if !store.Allow(limit.Key("198.51.100.4"), limit, base).Allowed {
		t.Fatal("one caller running out refused a different caller")
	}
}

/*
 * A replenished bucket is indistinguishable from an absent one, so discarding it
 * is free — and necessary, because the map is keyed on caller address and would
 * otherwise grow for as long as the process runs.
 */
func TestReplenishedEntriesAreDiscarded(t *testing.T) {
	store := throttle.NewMemoryStore(0)
	limit := perMinute(5)

	for index := range 50 {
		store.Allow(limit.Key(fmt.Sprintf("203.0.113.%d", index)), limit, base)
	}

	if store.Len() != 50 {
		t.Fatalf("tracking %d callers, want 50", store.Len())
	}

	// A full window later every one of those has refilled completely, so the
	// next request should find them gone rather than accumulating.
	store.Allow(limit.Key("198.51.100.4"), limit, base.Add(limit.Per+time.Second))

	if store.Len() != 1 {
		t.Fatalf("tracking %d callers after a full window, want 1", store.Len())
	}
}

/*
 * An entry mid-refill still carries state, so the sweep must leave it alone.
 *
 * The window is deliberately much longer than the interval the sweep runs on. If
 * the two matched, the only moment a sweep could happen would be one at which the
 * bucket had already replenished, and this test would pass no matter what the
 * sweep discarded.
 */
func TestPartiallyRefilledEntriesSurviveTheSweep(t *testing.T) {
	store := throttle.NewMemoryStore(0)
	limit := throttle.Limit{Name: "global", Requests: 10, Per: 10 * time.Minute}

	// Spend the lot, so replenishing takes the full ten minutes.
	for range limit.Requests {
		store.Allow(testKey, limit, base)
	}

	// Two minutes on: long enough for a sweep to run, far short of replenished.
	later := base.Add(2 * time.Minute)
	store.Allow(limit.Key("198.51.100.4"), limit, later)

	decision := store.Allow(testKey, limit, later)

	// Two of ten minutes returns two tokens; one is spent here, leaving one.
	if decision.Remaining != 1 {
		t.Fatalf(
			"remaining = %d, want 1: a swept entry would come back as a full allowance of %d",
			decision.Remaining, limit.Requests,
		)
	}
}

/*
 * The bound is a security requirement, not tidiness: keyed on caller address, an
 * unbounded map lets whoever rotates source addresses turn the throttle into the
 * memory exhaustion vector it exists to prevent.
 */
func TestTrackedCallersStayBounded(t *testing.T) {
	const maxKeys = 100

	store := throttle.NewMemoryStore(maxKeys)
	limit := perMinute(5)

	// Ten times the cap, all within one instant so nothing replenishes and the
	// lossless sweep cannot help.
	for index := range maxKeys * 10 {
		store.Allow(limit.Key(fmt.Sprintf("10.0.%d.%d", index/256, index%256)), limit, base)
	}

	if store.Len() > maxKeys {
		t.Fatalf("tracking %d callers, which exceeds the cap of %d", store.Len(), maxKeys)
	}
}

// Eviction must drop the stalest entries, not whichever the map iterated to
// first: an active caller losing its bucket gets a fresh allowance.
func TestEvictionDropsTheStalestCaller(t *testing.T) {
	const maxKeys = 10

	store := throttle.NewMemoryStore(maxKeys)
	limit := throttle.Limit{Name: "global", Requests: 2, Per: time.Hour}

	// Fill the store, oldest first, one second apart.
	for index := range maxKeys {
		store.Allow(limit.Key(fmt.Sprintf("10.0.0.%d", index)), limit, base.Add(time.Duration(index)*time.Second))
	}

	recent := limit.Key(fmt.Sprintf("10.0.0.%d", maxKeys-1))
	later := base.Add(time.Duration(maxKeys) * time.Second)

	// Force an eviction with a new caller. A window of an hour means nothing has
	// replenished, so this cannot be satisfied by the lossless sweep.
	store.Allow(limit.Key("198.51.100.4"), limit, later)

	// The most recently active caller must still be mid-allowance.
	decision := store.Allow(recent, limit, later)
	if decision.Remaining != 0 {
		t.Fatalf(
			"remaining = %d for the most recently active caller, want 0: its bucket was evicted ahead of staler ones",
			decision.Remaining,
		)
	}
}

/*
 * A limit that permits nothing must refuse rather than divide by its own rate.
 * Unreachable through configuration, where a non-positive limit leaves the
 * middleware uninstalled, but a store returning an infinite retry would be a
 * worse failure than one that says no.
 */
func TestNonPositiveLimitRefusesWithoutDividingByZero(t *testing.T) {
	store := throttle.NewMemoryStore(0)

	for _, limit := range []throttle.Limit{
		{Name: "global", Requests: 0, Per: time.Minute},
		{Name: "global", Requests: -1, Per: time.Minute},
		{Name: "global", Requests: 10, Per: 0},
	} {
		decision := store.Allow(testKey, limit, base)

		if decision.Allowed {
			t.Errorf("limit %+v allowed a request", limit)
		}

		if decision.RetryAfter < 0 {
			t.Errorf("limit %+v produced a negative retry after: %s", limit, decision.RetryAfter)
		}
	}
}

// A clock stepping backwards must not remove tokens a caller already earned.
func TestTimeMovingBackwardsDoesNotTakeTokensAway(t *testing.T) {
	store := throttle.NewMemoryStore(0)
	limit := perMinute(5)

	store.Allow(testKey, limit, base)

	decision := store.Allow(testKey, limit, base.Add(-time.Hour))
	if !decision.Allowed {
		t.Fatal("a request was refused because the clock went backwards")
	}

	if decision.Remaining != limit.Requests-2 {
		t.Fatalf("remaining = %d, want %d", decision.Remaining, limit.Requests-2)
	}
}

/*
 * Concurrent callers must not be able to overspend a shared allowance. Run under
 * -race in CI; the count assertion is what catches a lost update that the race
 * detector would not see.
 */
func TestConcurrentRequestsCannotExceedTheAllowance(t *testing.T) {
	store := throttle.NewMemoryStore(0)
	limit := perMinute(50)

	const attempts = 500

	var (
		waiting sync.WaitGroup
		mutex   sync.Mutex
		allowed int
	)

	for range attempts {
		waiting.Add(1)

		go func() {
			defer waiting.Done()

			// One fixed instant for every goroutine, so no token can refill
			// mid-test and the expected total is exactly the burst.
			if store.Allow(testKey, limit, base).Allowed {
				mutex.Lock()
				allowed++
				mutex.Unlock()
			}
		}()
	}

	waiting.Wait()

	if allowed != limit.Requests {
		t.Fatalf("%d of %d concurrent requests were allowed, want exactly %d", allowed, attempts, limit.Requests)
	}
}
