package throttle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
	"web-app/app/container"
	"web-app/app/http/middlewares"
	"web-app/app/services/throttle"
	"web-app/configs"
	"web-app/tests/support"

	"github.com/gin-gonic/gin"
)

/*
 * These need no database: every request is either refused by the throttle or
 * rejected by validation before a handler reaches for a connection.
 *
 * They go through the real engine, which is the only way the placement
 * assertions mean anything — where the throttle sits relative to the exception
 * handler and to CORS is decided in HTTPServiceProvider.GlobalMiddleware.
 */

// envelope is the four-key response shape every endpoint answers with.
type envelope struct {
	Status  string              `json:"status"`
	Message string              `json:"message"`
	Data    any                 `json:"data"`
	Errors  map[string][]string `json:"errors"`
}

/*
 * routerWith builds the application with explicit allowances.
 *
 * Limits are passed rather than read from the environment: process environment is
 * global and unordered with respect to other tests, so a test that set it would
 * leak into whatever ran next.
 *
 * @param global Requests allowed globally; zero leaves the throttle uninstalled.
 * @param login  Requests allowed on the login route.
 */
func routerWith(t *testing.T, global, login int) *gin.Engine {
	t.Helper()

	router, _ := support.AppRouterWith(t, func(config *container.Config) {
		config.Throttle = &configs.ThrottleConfig{
			Store:   configs.MemoryStoreDriver,
			Global:  throttle.Limit{Name: "global", Requests: global, Per: time.Minute},
			Login:   throttle.Limit{Name: "login", Requests: login, Per: time.Minute},
			MaxKeys: throttle.DefaultMaxKeys,
		}
	})

	return router
}

func get(t *testing.T, router *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	return recorder
}

// exhaust spends the whole allowance on a path and returns the response that
// follows it.
func exhaust(t *testing.T, router *gin.Engine, path string, allowance int) *httptest.ResponseRecorder {
	t.Helper()

	for attempt := 1; attempt <= allowance; attempt++ {
		if recorder := get(t, router, path); recorder.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d of an allowance of %d was refused", attempt, allowance)
		}
	}

	return get(t, router, path)
}

/*
 * The placement guard, and the reason this test exists at all.
 *
 * The exception handler runs ctx.Next() and inspects the outcome on the way out,
 * so a middleware that aborts above it is never observed: the 429 would not
 * render and gin would answer with an empty 200. Asserting on the body rather
 * than only the status is what catches that, because the status alone is
 * unset — and therefore 200 — in exactly the broken case.
 */
func TestRefusalRendersTheEnvelope(t *testing.T) {
	const allowance = 3

	router := routerWith(t, allowance, 5)
	recorder := exhaust(t, router, "/", allowance)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
	}

	var body envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("the refusal carried no envelope (%v); body: %q", err, recorder.Body.String())
	}

	if body.Status != "error" {
		t.Errorf("status = %q, want %q", body.Status, "error")
	}

	if body.Message == "" {
		t.Error("the refusal carried no message")
	}

	if body.Data != nil {
		t.Errorf("data = %v on a refusal, want null", body.Data)
	}
}

/*
 * A refusal has to say how long to wait, and the number has to be no shorter
 * than the real wait or the client returns and is refused again.
 *
 * Seven per minute is chosen so the true wait is fractional — 60/7 is about 8.57
 * seconds — which is what makes this able to tell rounding up from truncating.
 * A whole-number allowance like 2 per minute divides exactly and would pass
 * either way.
 */
func TestRefusalCarriesRetryAfter(t *testing.T) {
	const allowance = 7

	router := routerWith(t, allowance, 5)
	recorder := exhaust(t, router, "/", allowance)

	retryAfter := recorder.Header().Get(middlewares.RetryAfterHeader)
	if retryAfter == "" {
		t.Fatal("the refusal carried no Retry-After header")
	}

	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		t.Fatalf("Retry-After %q is not an integer number of seconds: %v", retryAfter, err)
	}

	const wantSeconds = 9 // ceil(60/7)

	if seconds != wantSeconds {
		t.Fatalf(
			"Retry-After = %d, want %d: a shorter value sends the caller back before a token exists",
			seconds, wantSeconds,
		)
	}
}

/*
 * Headers on successful responses are what let a client pace itself. Without
 * them the only way to discover the allowance is to exceed it.
 */
func TestSuccessfulResponsesCarryTheAllowance(t *testing.T) {
	const allowance = 10

	router := routerWith(t, allowance, 5)

	first := get(t, router, "/")
	if first.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", first.Code, http.StatusOK)
	}

	if got := first.Header().Get(middlewares.RateLimitLimitHeader); got != strconv.Itoa(allowance) {
		t.Errorf("%s = %q, want %q", middlewares.RateLimitLimitHeader, got, strconv.Itoa(allowance))
	}

	if got := first.Header().Get(middlewares.RateLimitResetHeader); got == "" {
		t.Errorf("%s was not set", middlewares.RateLimitResetHeader)
	}

	firstRemaining := first.Header().Get(middlewares.RateLimitRemainingHeader)
	if firstRemaining != strconv.Itoa(allowance-1) {
		t.Errorf("remaining = %q after one request, want %q", firstRemaining, strconv.Itoa(allowance-1))
	}

	second := get(t, router, "/")
	if got := second.Header().Get(middlewares.RateLimitRemainingHeader); got != strconv.Itoa(allowance-2) {
		t.Errorf("remaining = %q after two requests, want %q", got, strconv.Itoa(allowance-2))
	}
}

/*
 * The per-route override. Login is refused well before the global allowance is
 * touched, because each attempt there costs an argon2id verification.
 */
func TestLoginIsRefusedSoonerThanTheGlobalAllowance(t *testing.T) {
	const (
		globalAllowance = 100
		loginAllowance  = 2
	)

	router := routerWith(t, globalAllowance, loginAllowance)

	for attempt := 1; attempt <= loginAllowance; attempt++ {
		recorder := support.PostJSON(t, router, support.V1Path("/login"), `{}`)
		if recorder.Code == http.StatusTooManyRequests {
			t.Fatalf("login attempt %d of %d was refused", attempt, loginAllowance)
		}
	}

	recorder := support.PostJSON(t, router, support.V1Path("/login"), `{}`)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf(
			"login attempt %d: status = %d, want %d; the route's own allowance was not applied",
			loginAllowance+1, recorder.Code, http.StatusTooManyRequests,
		)
	}
}

/*
 * The two allowances are counted separately, which is what makes applying both
 * to one request safe. Sharing a counter would make each one tighter than
 * configured, and exhausting the login limit would lock a caller out of the
 * whole API.
 */
func TestExhaustingTheLoginAllowanceLeavesTheGlobalOneIntact(t *testing.T) {
	const (
		globalAllowance = 20
		loginAllowance  = 2
	)

	router := routerWith(t, globalAllowance, loginAllowance)

	for range loginAllowance + 1 {
		support.PostJSON(t, router, support.V1Path("/login"), `{}`)
	}

	recorder := get(t, router, "/")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: the login allowance drew from the global one", recorder.Code, http.StatusOK)
	}

	/*
	 * The login attempts each spent a global token too, since the global
	 * throttle applies to every route. What must not have happened is the login
	 * limit consuming the global allowance beyond its own request count.
	 */
	remaining, err := strconv.Atoi(recorder.Header().Get(middlewares.RateLimitRemainingHeader))
	if err != nil {
		t.Fatalf("remaining is not an integer: %v", err)
	}

	if want := globalAllowance - (loginAllowance + 1) - 1; remaining != want {
		t.Fatalf("global remaining = %d, want %d", remaining, want)
	}
}

/*
 * A browser sends a preflight before every non-simple cross-origin request. If
 * those spent tokens, a browser client would consume its allowance twice as
 * fast as it makes requests — so the throttle sits below CORS, which answers
 * and ends a preflight before it gets here.
 */
func TestPreflightRequestsDoNotSpendTheAllowance(t *testing.T) {
	const (
		allowance = 3
		origin    = "https://app.example.com"
	)

	router, _ := support.AppRouterWith(t, func(config *container.Config) {
		config.App.URL = "https://api.example.com"
		config.App.CORSOrigins = []string{origin}
		config.Throttle = &configs.ThrottleConfig{
			Store:   configs.MemoryStoreDriver,
			Global:  throttle.Limit{Name: "global", Requests: allowance, Per: time.Minute},
			Login:   throttle.Limit{Name: "login", Requests: 5, Per: time.Minute},
			MaxKeys: throttle.DefaultMaxKeys,
		}
	})

	// Comfortably more preflights than the whole allowance.
	for range allowance * 3 {
		request := httptest.NewRequest(http.MethodOptions, support.V1Path("/events"), nil)
		request.Header.Set("Origin", origin)
		request.Header.Set("Access-Control-Request-Method", http.MethodPost)

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		if recorder.Code == http.StatusTooManyRequests {
			t.Fatal("a preflight was refused by the throttle")
		}
	}

	recorder := get(t, router, "/")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: preflights consumed the allowance", recorder.Code, http.StatusOK)
	}
}

/*
 * A limit of zero installs nothing at all rather than refusing everything, the
 * same way an empty origin list installs no CORS. A configuration that could
 * only be read as "allow none" would take the application down on a typo.
 */
func TestAZeroLimitInstallsNoThrottle(t *testing.T) {
	router := routerWith(t, 0, 5)

	for attempt := range 40 {
		recorder := get(t, router, "/")

		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", attempt+1, recorder.Code, http.StatusOK)
		}

		if got := recorder.Header().Get(middlewares.RateLimitLimitHeader); got != "" {
			t.Fatalf("a disabled throttle still advertised an allowance: %q", got)
		}
	}
}

// Callers are counted separately, so one exhausting its allowance must not
// refuse another.
func TestOneCallerCannotExhaustAnother(t *testing.T) {
	const allowance = 2

	router := routerWith(t, allowance, 5)

	exhausted := exhaust(t, router, "/", allowance)
	if exhausted.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", exhausted.Code, http.StatusTooManyRequests)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "198.51.100.4:54321"

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("a second caller got %d, want %d: the allowance is not keyed per caller", recorder.Code, http.StatusOK)
	}
}
