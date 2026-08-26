/*
 * Package throttle counts requests against named allowances, mirroring
 * Laravel's RateLimiter.
 *
 * It knows nothing about HTTP: a caller supplies a key, an allowance and the
 * current time, and gets back a decision plus the numbers a response needs. That
 * keeps the arithmetic testable without a server and without sleeping, and it is
 * why the current time is a parameter rather than something read inside.
 */
package throttle

import "time"

/*
 * Limit is a named allowance of Requests per Per.
 *
 * The name is a keyspace rather than a label. Two limits applied to one request
 * — a loose global one and a tight one on a single route — must draw down
 * separate counters, or each would silently consume the other's budget and both
 * would be tighter than configured.
 */
type Limit struct {
	Name     string
	Requests int
	Per      time.Duration
}

/*
 * Key namespaces an identity under this limit.
 *
 * @param identity The caller being counted, usually a client address.
 * @return string The store key.
 */
func (limit Limit) Key(identity string) string {
	return limit.Name + "|" + identity
}

/*
 * refillPerSecond is the rate tokens return at.
 *
 * A limit of 60 per minute refills at one per second while allowing a burst of
 * 60, which is the same steady state as a fixed window without the doubled burst
 * a window boundary permits.
 *
 * @return float64 Tokens per second, or zero if the limit permits nothing.
 */
func (limit Limit) refillPerSecond() float64 {
	if limit.Per <= 0 || limit.Requests <= 0 {
		return 0
	}

	return float64(limit.Requests) / limit.Per.Seconds()
}

/*
 * Decision is the outcome of one Allow call.
 *
 * It carries every number the response headers need so that the caller performs
 * no arithmetic of its own — the rate limiting logic stays in one place.
 */
type Decision struct {
	Allowed bool

	// Limit and Remaining are the allowance and what is left of it after this
	// request, for X-RateLimit-Limit and X-RateLimit-Remaining.
	Limit     int
	Remaining int

	// RetryAfter is how long until one token is available. Zero when allowed.
	RetryAfter time.Duration

	// ResetAt is when the allowance is fully replenished.
	ResetAt time.Time
}

/*
 * Store decides whether a key may spend one request against a limit.
 *
 * A single small method on purpose: it is the seam a shared implementation
 * backed by Redis drops into when a second replica appears, without any route or
 * middleware changing. The in-memory implementation counts per process, so N
 * replicas permit N times the configured limit.
 */
type Store interface {
	Allow(key string, limit Limit, now time.Time) Decision
}
