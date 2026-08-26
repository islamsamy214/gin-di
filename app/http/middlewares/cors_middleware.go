package middlewares

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

/*
 * WildcardOrigin permits any origin.
 *
 * Only sound because credentials are not allowed (see corsAllowCredentials): the
 * CORS specification forbids pairing a wildcard origin with credentials, and a
 * browser rejects the combination outright.
 */
const WildcardOrigin = "*"

/*
 * corsAllowCredentials keeps cookies and HTTP auth out of cross-origin requests.
 *
 * Authentication here is a bearer token in the Authorization header, which is a
 * plain allowed header rather than a credential, so nothing needs this on. It
 * stays off deliberately: turning it on makes the browser send cookies with
 * cross-origin requests, which is the precondition for CSRF, and it is invalid
 * alongside WildcardOrigin.
 */
const corsAllowCredentials = false

// corsMaxAge is how long a browser may cache a preflight result, capping how
// often an OPTIONS round trip precedes a real request.
const corsMaxAge = 12 * time.Hour

/*
 * corsAllowedMethods and corsAllowedHeaders describe the API, not the
 * deployment, so they are code rather than environment. Only the origin list
 * changes between a laptop and production, and that is the one thing read from
 * the environment.
 */
var (
	corsAllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsAllowedHeaders = []string{"Authorization", "Content-Type"}
)

/*
 * CORS answers browser preflights and marks responses as readable by the
 * configured origins.
 *
 * This is a browser control and not an authorization one. It cannot stop curl, a
 * server-side client or a script, because those never ask: only a browser
 * volunteers an Origin header and then withholds the response from the calling
 * page. The API's actual access control is the auth middleware, and CORS being
 * permissive does not weaken it.
 *
 * A request carrying no Origin header passes through entirely untouched, which is
 * what makes installing this safe for the existing non-browser clients.
 *
 * One deliberate exception to the response envelope: gin-contrib answers a
 * disallowed origin with a bare 403 and an empty body, and exposes no hook to
 * change it. That is left alone rather than worked around. The response is only
 * reachable for a request that carried an Origin header — a browser — and a
 * browser discards the body of a failed CORS check without exposing it to the
 * calling page, so an envelope there would be unobservable by construction.
 * Pinned by tests/Feature/cors so a library change is visible.
 *
 * @param origins The allowed browser origins, or a single WildcardOrigin.
 * @return gin.HandlerFunc The middleware.
 * @return error           If an origin could never match what a browser sends.
 */
func CORS(origins []string) (gin.HandlerFunc, error) {
	if err := validateOrigins(origins); err != nil {
		return nil, err
	}

	settings := cors.Config{
		AllowMethods: corsAllowedMethods,
		AllowHeaders: corsAllowedHeaders,

		/*
		 * A browser can read the status and body of a cross-origin response but
		 * none of its other headers unless they are named here.
		 *
		 * The correlation id is what ties a client-side error report back to a
		 * line in the log. The rate limit headers are what let a browser client
		 * pace itself and honour a refusal — without them a throttled frontend
		 * sees a 429 with no idea how long to wait, which is the situation the
		 * headers exist to prevent.
		 */
		ExposeHeaders: []string{
			RequestIDHeader,
			RateLimitLimitHeader,
			RateLimitRemainingHeader,
			RateLimitResetHeader,
			RetryAfterHeader,
		},

		AllowCredentials: corsAllowCredentials,
		MaxAge:           corsMaxAge,
	}

	/*
	 * AllowAllOrigins and AllowOrigins are mutually exclusive in gin-contrib:
	 * setting both panics. The wildcard is expressed as the former.
	 */
	if len(origins) == 1 && origins[0] == WildcardOrigin {
		settings.AllowAllOrigins = true
	} else {
		settings.AllowOrigins = origins
	}

	return cors.New(settings), nil
}

/*
 * EffectiveOrigins is the allowlist the middleware is actually built from: the
 * configured origins plus this application's own.
 *
 * The application's own origin has to be on the list, and not because a browser
 * would ever ask about it cross-origin. A browser sends Origin on same-origin
 * requests too whenever the method is not GET or HEAD, so a same-origin POST
 * arrives carrying the API's own origin — and gin-contrib refuses any origin not
 * on the list with a hard 403. Leaving it off means a frontend served from the
 * same host as this API can read but cannot write, with a 403 that mentions
 * nothing about CORS.
 *
 * The application URL is normalised rather than required to be exact, because it
 * is configured for humans and legitimately carries a trailing slash or a path
 * that an Origin header never has.
 *
 * @param appURL     The configured application URL.
 * @param configured The origins read from the environment.
 * @return []string The effective allowlist, empty when CORS is disabled.
 * @return error    If CORS is enabled and the application URL is not usable.
 */
func EffectiveOrigins(appURL string, configured []string) ([]string, error) {
	// Empty means no browser client, so no middleware and nothing to merge into.
	if len(configured) == 0 {
		return nil, nil
	}

	/*
	 * A wildcard already admits this application's own origin, and adding it
	 * alongside would trip the rule against mixing the two. The URL is not even
	 * parsed here, so a deployment that opts into the wildcard is not held to
	 * having a valid one.
	 */
	if len(configured) == 1 && configured[0] == WildcardOrigin {
		return configured, nil
	}

	self, err := originFrom(appURL)
	if err != nil {
		return nil, fmt.Errorf("cors origins: deriving this application's own origin: %w", err)
	}

	origins := make([]string, 0, len(configured)+1)
	origins = append(origins, self)

	for _, origin := range configured {
		if !strings.EqualFold(origin, self) {
			origins = append(origins, origin)
		}
	}

	return origins, nil
}

/*
 * originFrom reduces a configured URL to the origin a browser would send.
 *
 * Scheme and host only, lowercased: a path, a trailing slash, a query or
 * credentials are all dropped rather than rejected, since none of them can
 * appear in an Origin header and their presence in an application URL is
 * ordinary.
 *
 * @param rawURL The configured URL.
 * @return string The origin.
 * @return error If there is no scheme and host to derive one from.
 */
func originFrom(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", errors.New("the application URL is empty; set APP_URL to this API's own base URL")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("%q is not a valid URL: %w", rawURL, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf(
			"%q needs an http:// or https:// scheme to yield a browser origin",
			rawURL,
		)
	}

	if parsed.Host == "" {
		return "", fmt.Errorf("%q has no host", rawURL)
	}

	return scheme + "://" + strings.ToLower(parsed.Host), nil
}

/*
 * validateOrigins rejects entries a browser could never match.
 *
 * A browser sends an origin as scheme://host[:port] — no path, no trailing
 * slash, no credentials, lowercase scheme. gin-contrib compares the Origin
 * header literally, so "http://localhost/" or a bare "localhost" is not a
 * permissive setting, it is a silently dead one: every request is refused and
 * the misconfiguration surfaces only as an opaque browser console error. Failing
 * at boot instead, exactly as an unparseable trusted proxy does.
 *
 * @param origins The configured origins.
 * @return error Describing the first entry that cannot match.
 */
func validateOrigins(origins []string) error {
	for _, origin := range origins {
		if origin == WildcardOrigin {
			if len(origins) > 1 {
				return fmt.Errorf(
					"cors origins: %q cannot be combined with named origins %v",
					WildcardOrigin, origins,
				)
			}

			continue
		}

		if err := validateOrigin(origin); err != nil {
			return fmt.Errorf("cors origins: %w", err)
		}
	}

	return nil
}

// validateOrigin checks a single entry has the exact shape of a browser Origin
// header.
func validateOrigin(origin string) error {
	parsed, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("%q is not a valid origin: %w", origin, err)
	}

	switch {
	case parsed.Scheme != "http" && parsed.Scheme != "https":
		return fmt.Errorf(
			"%q must start with http:// or https:// — a browser origin always carries a scheme",
			origin,
		)
	case parsed.Host == "":
		return fmt.Errorf("%q is missing a host", origin)
	case parsed.Path != "":
		return fmt.Errorf(
			"%q must not include a path or trailing slash: use %s://%s",
			origin, parsed.Scheme, parsed.Host,
		)
	case parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil:
		return fmt.Errorf("%q must be only a scheme, host and optional port", origin)
	case origin != strings.ToLower(origin):
		return fmt.Errorf("%q must be lowercase to match the Origin header a browser sends", origin)
	}

	return nil
}
