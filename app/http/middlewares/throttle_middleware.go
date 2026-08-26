package middlewares

import (
	"math"
	"strconv"
	"time"
	"web-app/app/exceptions"
	"web-app/app/services/throttle"

	"github.com/gin-gonic/gin"
)

// The rate limit headers, matching the names Laravel emits so existing clients
// need no new vocabulary. RetryAfterHeader is the standard one from RFC 9110.
const (
	RateLimitLimitHeader     = "X-RateLimit-Limit"
	RateLimitRemainingHeader = "X-RateLimit-Remaining"
	RateLimitResetHeader     = "X-RateLimit-Reset"
	RetryAfterHeader         = "Retry-After"
)

/*
 * Throttle refuses a caller that has spent its allowance, mirroring Laravel's
 * throttle middleware.
 *
 * Applied globally for the default allowance and again on a group or a single
 * route for a tighter one. Both are safe together because Limit.Key namespaces
 * the counter: without that, the tight limit and the loose one would draw down
 * the same tokens and each would be stricter than configured.
 *
 * Keyed on ClientIP(), which is only meaningful because Engine configures
 * SetTrustedProxies. Untrusted, a caller picks its own X-Forwarded-For and gets
 * a fresh allowance per request, which is why this was not worth adding before
 * that existed. Note the opposite misconfiguration is an outage rather than a
 * hole: with a proxy in front that is not trusted, every request appears to come
 * from the proxy, so the whole application shares one allowance and throttles as
 * a single client.
 *
 * Keying authenticated routes on user id instead of address is a deliberate
 * omission — it matters for callers behind NAT, and it is a small change here
 * when wanted.
 *
 * @param store The counter, shared across every limit.
 * @param limit The allowance this route or group draws from.
 * @return gin.HandlerFunc The middleware.
 */
func Throttle(store throttle.Store, limit throttle.Limit) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		decision := store.Allow(limit.Key(ctx.ClientIP()), limit, time.Now())

		/*
		 * Set on success as well as refusal. A client that can see what is left
		 * can pace itself; one that cannot has to discover the limit by being
		 * refused, which is the behaviour the headers exist to avoid.
		 */
		header := ctx.Writer.Header()
		header.Set(RateLimitLimitHeader, strconv.Itoa(decision.Limit))
		header.Set(RateLimitRemainingHeader, strconv.Itoa(decision.Remaining))
		header.Set(RateLimitResetHeader, strconv.FormatInt(decision.ResetAt.Unix(), 10))

		if decision.Allowed {
			ctx.Next()

			return
		}

		/*
		 * Written before the error is reported, because the exception carries no
		 * headers: the handler renders a body from the exception's status and
		 * message, and anything already on the writer survives that.
		 *
		 * Rounded up, and never below one. Retry-After is integer seconds, so
		 * rounding to nearest would tell a caller waiting 1.4s to return after
		 * 1s and be refused again; a zero would invite an immediate retry with
		 * the same result.
		 */
		retryAfter := int(math.Ceil(decision.RetryAfter.Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}

		header.Set(RetryAfterHeader, strconv.Itoa(retryAfter))

		_ = ctx.Error(exceptions.NewTooManyRequests())
		ctx.Abort()
	}
}
