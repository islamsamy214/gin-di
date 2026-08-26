package support

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"web-app/app/container"
	"web-app/app/providers"
	"web-app/app/services"
	"web-app/app/services/core"
	"web-app/configs"
	httpApis "web-app/routes/http"

	"github.com/gin-gonic/gin"
)

/*
 * AppRouter mounts the real engine, so tests exercise the same wiring, middleware
 * stack and fallbacks that Boot builds rather than a stand-in.
 *
 * It goes through HTTPServiceProvider.Engine rather than assembling a router by
 * hand: the response envelope, the exception handler and the 404/405 handlers all
 * live there, so a hand-rolled router would leave every one of them untested.
 *
 * @return *gin.Engine            The router.
 * @return *services.AuthService  The service its middleware was built with.
 */
func AppRouter(t *testing.T) (*gin.Engine, *services.AuthService) {
	t.Helper()

	router, resolved := AppRouterWithContainer(t)

	return router, resolved.Auth()
}

/*
 * AppRouterWithContainer returns the router alongside the container it was built
 * from, for tests that need the connection or the logger as well.
 */
func AppRouterWithContainer(t *testing.T) (*gin.Engine, *container.Container) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	// The validation rules are process-global on gin's shared validator, and the
	// requests bound below carry tags that do not exist until this runs.
	if err := providers.NewValidationServiceProvider().Boot(); err != nil {
		t.Fatalf("booting validation: %v", err)
	}

	// Discarded rather than written: a passing test should not append to
	// storage/logs, and a failing one reports through t instead.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	db := testConnection(t)

	resolved := container.New(container.Config{
		App:    configs.NewAppConfig(),
		Auth:   AuthService(t, TestSecret),
		Users:  services.NewUserService(db),
		DB:     db,
		Logger: logger,
	})

	router, err := providers.NewHTTPServiceProvider().Engine(resolved)
	if err != nil {
		t.Fatalf("building the engine: %v", err)
	}

	return router, resolved
}

/*
 * testConnection resolves the shared pool without failing a test that has no
 * database.
 *
 * Route tests that never touch a handler needing the database — the 404, 405 and
 * unauthenticated cases — must still be able to build the engine. Those that do
 * touch it call FreshDatabase, which skips on its own when nothing is reachable.
 */
func testConnection(t *testing.T) *core.PostgresService {
	t.Helper()

	db, err := core.Connection()
	if err != nil {
		return nil
	}

	return db
}

// V1Path prefixes a route with the v1 API mount point, so a prefix change stays
// a one-line edit rather than a sweep through every suite.
func V1Path(route string) string {
	return httpApis.APIPrefix + "/v1" + route
}

/*
 * Request issues an arbitrary request against the router.
 *
 * The single implementation the other helpers delegate to, so the recorder and
 * header handling exist once.
 *
 * @param method        The HTTP method.
 * @param path          The request path.
 * @param body          The request body; empty sends none.
 * @param authorization The Authorization header; empty sends none.
 */
func Request(t *testing.T, router *gin.Engine, method, path, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	request := httptest.NewRequest(method, path, reader)

	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

// PostJSON sends a JSON body to the given path.
func PostJSON(t *testing.T, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	return Request(t, router, http.MethodPost, path, body, "")
}

// PostJSONAs sends a JSON body as an authenticated caller.
func PostJSONAs(t *testing.T, router *gin.Engine, path, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()

	return Request(t, router, http.MethodPost, path, body, authorization)
}

// GetPath issues a GET, optionally carrying an Authorization header.
func GetPath(t *testing.T, router *gin.Engine, path, authorization string) *httptest.ResponseRecorder {
	t.Helper()

	return Request(t, router, http.MethodGet, path, "", authorization)
}
