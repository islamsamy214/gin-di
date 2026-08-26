package cors

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"web-app/app/container"
	"web-app/app/http/middlewares"
	"web-app/app/providers"
	"web-app/app/services/throttle"
	"web-app/configs"
	"web-app/tests/support"

	"github.com/gin-gonic/gin"
)

/*
 * These need no database: every request either is answered by the CORS
 * middleware itself or is rejected by the auth middleware before a handler runs.
 *
 * They go through the real engine, which is the only way the ordering assertions
 * mean anything — where CORS sits relative to the exception handler is decided in
 * HTTPServiceProvider.GlobalMiddleware, not here.
 */

const allowedOrigin = "https://app.example.com"

const accessControlAllowOrigin = "Access-Control-Allow-Origin"

const appURL = "https://api.example.com"

// routerWithSelf builds the application with both its own URL and a CORS list,
// which is the arrangement EffectiveOrigins merges.
func routerWithSelf(t *testing.T, url string, origins ...string) *gin.Engine {
	t.Helper()

	router, _ := support.AppRouterWith(t, func(config *container.Config) {
		config.App.URL = url
		config.App.CORSOrigins = origins
	})

	return router
}

/*
 * routerWithOrigins builds the application with a CORS origin list, standing in
 * for APP_CORS_ORIGINS without touching process environment.
 *
 * The application URL is set explicitly because it has no default: enabling CORS
 * without one is a boot error, which is exactly what a real deployment faces.
 */
func routerWithOrigins(t *testing.T, origins ...string) *gin.Engine {
	t.Helper()

	return routerWithSelf(t, appURL, origins...)
}

// preflight sends the OPTIONS request a browser sends before a cross-origin
// POST.
func preflight(t *testing.T, router *gin.Engine, origin, path string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodOptions, path, nil)
	request.Header.Set("Origin", origin)
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

/*
 * The default has to stay silent. A server-to-server client never sends Origin,
 * so an API with no browser consumer should not advertise a cross-origin policy
 * at all — and this is what guarantees adding CORS changed nothing for the
 * existing deployment.
 */
func TestNoOriginsConfiguredEmitsNoCORSHeaders(t *testing.T) {
	router := routerWithOrigins(t)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", allowedOrigin)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(accessControlAllowOrigin); got != "" {
		t.Fatalf("unconfigured CORS still advertised an allowed origin: %q", got)
	}
}

func TestAllowedOriginIsEchoedBack(t *testing.T) {
	router := routerWithOrigins(t, allowedOrigin)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", allowedOrigin)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(accessControlAllowOrigin); got != allowedOrigin {
		t.Fatalf("allowed origin not echoed: got %q, want %q", got, allowedOrigin)
	}

	/*
	 * The correlation id is useless to a browser client unless it is exposed:
	 * without this header the fetch response object hides it, and a client-side
	 * error report cannot be tied back to a log line.
	 */
	if got := recorder.Header().Get("Access-Control-Expose-Headers"); got == "" {
		t.Fatal("the request id header was not exposed to the browser")
	}
}

/*
 * An origin outside the list must not be echoed. This is the assertion that
 * would catch a regression to origin reflection — the failure mode where the
 * middleware repeats whatever Origin it was sent and the allowlist becomes
 * decorative.
 */
func TestDisallowedOriginIsNotEchoed(t *testing.T) {
	router := routerWithOrigins(t, allowedOrigin)

	const attacker = "https://evil.example.com"

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", attacker)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(accessControlAllowOrigin); got == attacker {
		t.Fatalf("an unlisted origin was reflected back: %q", got)
	}
}

/*
 * OPTIONS is not a registered method on any route, and Engine sets
 * HandleMethodNotAllowed, so without the CORS middleware intercepting it a
 * preflight is answered with the 405 envelope and every cross-origin POST fails
 * before it is sent. This pins that interaction.
 */
func TestPreflightIsAnsweredRatherThanRejectedAs405(t *testing.T) {
	router := routerWithOrigins(t, allowedOrigin)

	recorder := preflight(t, router, allowedOrigin, support.V1Path("/events"))

	if recorder.Code == http.StatusMethodNotAllowed {
		t.Fatal("preflight was answered with 405; the CORS middleware did not intercept OPTIONS")
	}

	if recorder.Code != http.StatusNoContent && recorder.Code != http.StatusOK {
		t.Fatalf("unexpected preflight status: %d", recorder.Code)
	}

	if got := recorder.Header().Get(accessControlAllowOrigin); got != allowedOrigin {
		t.Fatalf("preflight did not allow the origin: got %q", got)
	}
}

func TestPreflightFromDisallowedOriginIsNotApproved(t *testing.T) {
	router := routerWithOrigins(t, allowedOrigin)

	recorder := preflight(t, router, "https://evil.example.com", support.V1Path("/events"))

	if recorder.Code == http.StatusNoContent && recorder.Header().Get(accessControlAllowOrigin) != "" {
		t.Fatal("preflight from an unlisted origin was approved")
	}
}

/*
 * A rejected request still has to be readable. The middleware sets its headers
 * on the way in, before the auth middleware aborts, so the 401 envelope written
 * on the way out carries them; if it did not, a browser would withhold the body
 * from the page and the client would see an opaque network error rather than the
 * reason it was rejected — the most common CORS symptom and the hardest to
 * diagnose from the client side.
 */
func TestErrorResponsesCarryCORSHeaders(t *testing.T) {
	router := routerWithOrigins(t, allowedOrigin)

	request := httptest.NewRequest(http.MethodGet, support.V1Path("/events"), nil)
	request.Header.Set("Origin", allowedOrigin)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected the request to be rejected as 401, got %d", recorder.Code)
	}

	if got := recorder.Header().Get(accessControlAllowOrigin); got != allowedOrigin {
		t.Fatalf("the 401 envelope was not readable cross-origin: allow-origin was %q", got)
	}
}

/*
 * The regression that matters most when switching CORS on: a client that sends no
 * Origin header at all — every server-side caller, curl, the health check — has to
 * be completely unaffected. If the middleware ever started rejecting or altering
 * those, enabling CORS for a browser client would silently break every other
 * consumer of the API.
 */
func TestRequestsWithoutAnOriginAreUntouched(t *testing.T) {
	router := routerWithOrigins(t, allowedOrigin)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("a request with no Origin was not served normally: got %d", recorder.Code)
	}

	if got := recorder.Header().Get(accessControlAllowOrigin); got != "" {
		t.Fatalf("a response to a request with no Origin carried CORS headers: %q", got)
	}

	if body := recorder.Body.String(); !strings.Contains(body, `"status"`) {
		t.Fatalf("the response envelope was altered: %s", body)
	}
}

/*
 * Pins gin-contrib's rejection of a disallowed origin: a bare 403 with no body,
 * which is the one response in the application that is not the four-key
 * envelope. Documented as deliberate in middlewares.CORS — this test exists so
 * that if a library upgrade changes the status or starts emitting a body, it is
 * noticed here rather than by a client.
 */
func TestDisallowedOriginIsRejectedWithoutAnEnvelope(t *testing.T) {
	router := routerWithOrigins(t, allowedOrigin)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://evil.example.com")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected a disallowed origin to be refused with 403, got %d", recorder.Code)
	}

	if body := recorder.Body.String(); body != "" {
		t.Fatalf("gin-contrib now returns a body on a CORS rejection: %q", body)
	}
}

/*
 * The application's own origin is allowed without being listed.
 *
 * This is what keeps a frontend served from the API's own host working: its
 * POSTs carry Origin, and gin-contrib refuses anything unlisted with a 403, so
 * without the merge a same-origin write fails for a reason that mentions neither
 * CORS nor the origin.
 */
func TestApplicationOwnOriginIsAllowedWithoutBeingListed(t *testing.T) {
	router := routerWithSelf(t, appURL, allowedOrigin)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", appURL)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("the application's own origin was refused: %d", recorder.Code)
	}

	if got := recorder.Header().Get(accessControlAllowOrigin); got != appURL {
		t.Fatalf("own origin not echoed: got %q, want %q", got, appURL)
	}
}

/*
 * The case above, as a browser actually produces it: a same-origin POST is the
 * only same-origin request that carries an Origin header, so it is the one that
 * would break.
 */
func TestSameOriginPostIsNotRefused(t *testing.T) {
	router := routerWithSelf(t, appURL, allowedOrigin)

	request := httptest.NewRequest(http.MethodPost, support.V1Path("/login"), strings.NewReader("{}"))
	request.Header.Set("Origin", appURL)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	// The empty body is rejected on its merits; what matters is that it reached
	// validation at all rather than being refused as a disallowed origin.
	if recorder.Code == http.StatusForbidden {
		t.Fatal("a same-origin POST was refused as a disallowed origin")
	}

	if got := recorder.Header().Get(accessControlAllowOrigin); got != appURL {
		t.Fatalf("same-origin POST did not carry CORS headers: %q", got)
	}
}

/*
 * Merging the application's own origin must not become a way in for everyone
 * else: the configured list still governs, and the addition is exactly one entry.
 */
func TestMergingOwnOriginDoesNotWidenTheAllowlist(t *testing.T) {
	router := routerWithSelf(t, appURL, allowedOrigin)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://evil.example.com")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("an unlisted origin was admitted after the merge: %d", recorder.Code)
	}
}

/*
 * With no origins configured the middleware is absent, so the application URL
 * must not switch CORS on by itself. Otherwise setting APP_URL — which every
 * deployment sets — would start refusing browser traffic with a 403 where it
 * previously passed through.
 */
func TestApplicationURLAloneDoesNotEnableCORS(t *testing.T) {
	router := routerWithSelf(t, appURL)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", appURL)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(accessControlAllowOrigin); got != "" {
		t.Fatalf("the application URL enabled CORS on its own: %q", got)
	}
}

/*
 * The application URL has no default, so an unset APP_URL cannot quietly become
 * a plausible-looking origin. A default without a port would not match a browser
 * on http://localhost:8000 — a port is part of an origin — and the symptom would
 * be a bare 403 on same-origin writes only.
 */
func TestApplicationURLHasNoDefault(t *testing.T) {
	if url := configs.NewAppConfig().URL; url != "" && url == "http://localhost" {
		t.Fatalf("APP_URL still falls back to a guessed origin: %q", url)
	}
}

/*
 * A port is part of an origin, so a URL that omits one does not admit a browser
 * that has one. This is the mistake the removed default used to make.
 */
func TestOriginComparisonIncludesThePort(t *testing.T) {
	origins, err := middlewares.EffectiveOrigins("http://localhost", []string{allowedOrigin})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, origin := range origins {
		if origin == "http://localhost:8000" {
			t.Fatal("a URL without a port produced an origin with one")
		}
	}

	if origins[0] != "http://localhost" {
		t.Fatalf("got %q, want %q", origins[0], "http://localhost")
	}
}

func TestEffectiveOrigins(t *testing.T) {
	cases := []struct {
		name       string
		appURL     string
		configured []string
		want       []string
	}{
		{
			name:   "disabled when nothing is configured",
			appURL: appURL,
		},
		{
			name:       "own origin is prepended",
			appURL:     appURL,
			configured: []string{allowedOrigin},
			want:       []string{appURL, allowedOrigin},
		},
		{
			// Otherwise the wildcard-mixed-with-named rule would reject it.
			name:       "wildcard is left alone",
			appURL:     appURL,
			configured: []string{middlewares.WildcardOrigin},
			want:       []string{middlewares.WildcardOrigin},
		},
		{
			name:       "a wildcard needs no usable application URL",
			appURL:     "",
			configured: []string{middlewares.WildcardOrigin},
			want:       []string{middlewares.WildcardOrigin},
		},
		{
			name:       "an already listed own origin is not duplicated",
			appURL:     appURL,
			configured: []string{appURL, allowedOrigin},
			want:       []string{appURL, allowedOrigin},
		},
		{
			name:       "a trailing slash is normalised away",
			appURL:     appURL + "/",
			configured: []string{allowedOrigin},
			want:       []string{appURL, allowedOrigin},
		},
		{
			name:       "a path is dropped",
			appURL:     appURL + "/api/v1",
			configured: []string{allowedOrigin},
			want:       []string{appURL, allowedOrigin},
		},
		{
			name:       "the port is kept, since it is part of the origin",
			appURL:     "http://localhost:8000/",
			configured: []string{allowedOrigin},
			want:       []string{"http://localhost:8000", allowedOrigin},
		},
		{
			name:       "case is normalised to match the Origin header",
			appURL:     "HTTPS://API.Example.com",
			configured: []string{allowedOrigin},
			want:       []string{appURL, allowedOrigin},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := middlewares.EffectiveOrigins(testCase.appURL, testCase.configured)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if strings.Join(got, ",") != strings.Join(testCase.want, ",") {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}

/*
 * An application URL that cannot yield an origin is a boot error once CORS is
 * enabled, rather than a silently missing entry that surfaces later as a 403 on
 * same-origin writes only.
 */
func TestUnusableApplicationURLFailsAtBootWhenCORSIsEnabled(t *testing.T) {
	cases := []struct {
		name   string
		appURL string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"no scheme", "api.example.com"},
		{"host and port with no scheme", "localhost:8000"},
		{"unsupported scheme", "ftp://api.example.com"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := middlewares.EffectiveOrigins(testCase.appURL, []string{allowedOrigin})
			if err == nil {
				t.Fatalf("%q was accepted as an application origin", testCase.appURL)
			}
		})
	}
}

// engineWithOrigins builds the engine directly, so a rejected configuration is
// observable as an error rather than a failed test.
func engineWithOrigins(origins []string) error {
	resolved := container.New(container.Config{
		App:    &configs.AppConfig{URL: appURL, CORSOrigins: origins},
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),

		// Engine consults these while assembling the middleware stack, so they
		// are dependencies even for a test that only cares about origins.
		Throttle: support.UnthrottledConfig(),
		Limiter:  throttle.NewMemoryStore(0),
	})

	_, err := providers.NewHTTPServiceProvider().Engine(resolved)

	return err
}

/*
 * Origins that can never match a browser's Origin header are a boot error, not a
 * silently closed door. Each of these is a real mistake: a URL with a path or
 * trailing slash, a bare host with no scheme, and a wildcard diluted with named
 * entries.
 */
func TestUnusableOriginsFailAtBoot(t *testing.T) {
	cases := []struct {
		name    string
		origins []string
	}{
		{"trailing slash", []string{"https://app.example.com/"}},
		{"with a path", []string{"https://app.example.com/api"}},
		{"no scheme", []string{"localhost"}},
		{"host and port with no scheme", []string{"localhost:3000"}},
		{"uppercase", []string{"https://App.Example.com"}},
		{"wildcard mixed with named", []string{middlewares.WildcardOrigin, allowedOrigin}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := engineWithOrigins(testCase.origins); err == nil {
				t.Fatalf("%v was accepted; it can never match an Origin header", testCase.origins)
			}
		})
	}
}

func TestUsableOriginsBoot(t *testing.T) {
	// Names are descriptive rather than the origin itself: go treats a slash in a
	// subtest name as a nesting separator, so a URL would register phantom
	// intermediate subtests that never run.
	cases := []struct {
		name    string
		origins []string
	}{
		{"local development host with a port", []string{"http://localhost:3000"}},
		{"https host", []string{"https://app.example.com"}},
		{"several origins", []string{"http://localhost:3000", "https://app.example.com"}},
		{"wildcard alone", []string{middlewares.WildcardOrigin}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := engineWithOrigins(testCase.origins); err != nil {
				t.Fatalf("%v was rejected: %v", testCase.origins, err)
			}
		})
	}
}
