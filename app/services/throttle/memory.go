package throttle

import (
	"math"
	"sort"
	"sync"
	"time"
)

// DefaultMaxKeys bounds how many distinct callers are tracked at once. See
// MemoryStore for why a bound is a security requirement rather than tidiness.
const DefaultMaxKeys = 10_000

/*
 * pruneInterval is how often the lossless sweep runs at most.
 *
 * Fixed rather than derived from a limit's window, because one store serves
 * several limits with different windows and the sweep must not be paced by
 * whichever one happened to arrive.
 */
const pruneInterval = time.Minute

/*
 * evictionLowWater is the fraction of the cap an eviction pass reduces to.
 *
 * Evicting one entry per insert would make every request at the cap walk the
 * whole map. Dropping a tenth at a time amortises that scan across the next
 * thousand inserts.
 */
const evictionLowWater = 0.9

/*
 * bucket is one caller's remaining allowance.
 *
 * fullAt is stored rather than recomputed so the sweep can decide whether an
 * entry is disposable without knowing which limit produced it.
 */
type bucket struct {
	tokens float64
	seenAt time.Time
	fullAt time.Time
}

/*
 * MemoryStore is a token bucket per key, held in this process.
 *
 * Bounded on purpose. A map keyed on client address grows with every distinct
 * caller, and an attacker rotating source addresses — trivial with an IPv6 /64 —
 * would otherwise turn the throttle into the memory exhaustion vector it exists
 * to prevent, which is the same failure the argon2 concurrency cap addresses.
 *
 * Two mechanisms keep it bounded. A fully replenished bucket is indistinguishable
 * from an absent one, so sweeping those away changes no future decision and is
 * free; that alone holds the map to roughly the callers active within one window.
 * A hard cap then backs it up for the case where new keys arrive faster than they
 * replenish.
 *
 * The sweep runs inline, so there is no background goroutine to own, cancel or
 * leak, and nothing to close. The cost is a rare pass over the map while the lock
 * is held, which only grows under the attack it defends against.
 */
type MemoryStore struct {
	mutex    sync.Mutex
	buckets  map[string]*bucket
	maxKeys  int
	prunedAt time.Time
}

/*
 * NewMemoryStore builds an empty store.
 *
 * @param maxKeys How many distinct callers to track; DefaultMaxKeys if not positive.
 * @return *MemoryStore The store.
 */
func NewMemoryStore(maxKeys int) *MemoryStore {
	if maxKeys <= 0 {
		maxKeys = DefaultMaxKeys
	}

	return &MemoryStore{
		buckets: make(map[string]*bucket),
		maxKeys: maxKeys,
	}
}

/*
 * Allow spends one request from key's allowance.
 *
 * @param key   The namespaced caller, from Limit.Key.
 * @param limit The allowance being spent.
 * @param now   The current time, supplied so the result is deterministic.
 * @return Decision Whether the request may proceed, and the numbers describing why.
 */
func (store *MemoryStore) Allow(key string, limit Limit, now time.Time) Decision {
	refill := limit.refillPerSecond()

	/*
	 * A limit that permits nothing cannot be spent from, and dividing by its
	 * rate would produce an infinite retry. Unreachable as configured — a
	 * non-positive limit leaves the middleware uninstalled — but a store that
	 * silently returned infinity would be worse than one that refuses.
	 */
	if refill <= 0 {
		return Decision{
			Limit:      limit.Requests,
			RetryAfter: limit.Per,
			ResetAt:    now.Add(limit.Per),
		}
	}

	burst := float64(limit.Requests)

	store.mutex.Lock()
	defer store.mutex.Unlock()

	store.sweep(now, false)

	entry, found := store.buckets[key]
	if !found {
		store.makeRoom(now)

		entry = &bucket{tokens: burst, seenAt: now}
		store.buckets[key] = entry
	} else if elapsed := now.Sub(entry.seenAt).Seconds(); elapsed > 0 {
		// Guarded rather than applied unconditionally: a clock stepping backwards
		// must not remove tokens a caller has already earned.
		entry.tokens = math.Min(burst, entry.tokens+elapsed*refill)
		entry.seenAt = now
	}

	decision := Decision{Limit: limit.Requests}

	if entry.tokens >= 1 {
		entry.tokens--
		decision.Allowed = true
	} else {
		// The shortfall is always under one token, so this is the wait until the
		// next one lands rather than until the allowance is whole again.
		decision.RetryAfter = secondsToDuration((1 - entry.tokens) / refill)
	}

	decision.Remaining = int(entry.tokens)
	decision.ResetAt = now.Add(secondsToDuration((burst - entry.tokens) / refill))
	entry.fullAt = decision.ResetAt

	return decision
}

/*
 * Len reports how many callers are currently tracked.
 *
 * Exists so a test can assert the map stays bounded; nothing in the request path
 * consults it.
 *
 * @return int The number of live buckets.
 */
func (store *MemoryStore) Len() int {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	return len(store.buckets)
}

/*
 * sweep discards buckets that have replenished completely.
 *
 * Lossless: such a bucket would be recreated identically on the caller's next
 * request, so removing it cannot let anyone exceed their allowance. Rate limited
 * to pruneInterval unless forced, so the ordinary request path does not walk the
 * map.
 *
 * Callers must hold the mutex.
 *
 * @param now    The current time.
 * @param forced Whether to run regardless of when the last sweep was.
 */
func (store *MemoryStore) sweep(now time.Time, forced bool) {
	if !forced && now.Sub(store.prunedAt) < pruneInterval {
		return
	}

	store.prunedAt = now

	for key, entry := range store.buckets {
		if !entry.fullAt.After(now) {
			delete(store.buckets, key)
		}
	}
}

/*
 * makeRoom keeps the map under its cap before a new key is inserted.
 *
 * Neither failing open nor failing closed is acceptable at the cap: the first
 * lets whoever floods the keyspace switch throttling off for everyone, the second
 * lets them deny service to everyone. Evicting the least recently active bounds
 * memory instead, and its worst case is a dormant caller receiving a fresh
 * allowance it could have had by waiting.
 *
 * Callers must hold the mutex.
 *
 * @param now The current time.
 */
func (store *MemoryStore) makeRoom(now time.Time) {
	if len(store.buckets) < store.maxKeys {
		return
	}

	// The lossless sweep first: if it frees space, nothing has to be discarded
	// that still carries state.
	store.sweep(now, true)

	if len(store.buckets) < store.maxKeys {
		return
	}

	type aged struct {
		key    string
		seenAt time.Time
	}

	entries := make([]aged, 0, len(store.buckets))
	for key, entry := range store.buckets {
		entries = append(entries, aged{key: key, seenAt: entry.seenAt})
	}

	sort.Slice(entries, func(first, second int) bool {
		return entries[first].seenAt.Before(entries[second].seenAt)
	})

	target := int(float64(store.maxKeys) * evictionLowWater)
	for index := 0; index < len(entries) && len(store.buckets) > target; index++ {
		delete(store.buckets, entries[index].key)
	}
}

/*
 * secondsToDuration converts fractional seconds to a duration.
 *
 * Rounded up, and never negative: a caller told to retry in zero time would
 * retry immediately and be refused again.
 *
 * @param seconds The interval in seconds.
 * @return time.Duration The interval, at least zero.
 */
func secondsToDuration(seconds float64) time.Duration {
	if seconds <= 0 {
		return 0
	}

	return time.Duration(math.Ceil(seconds * float64(time.Second)))
}
