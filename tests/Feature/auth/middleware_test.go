package auth

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"web-app/app/http/middlewares"
	"web-app/app/services"
	"web-app/tests/support"

	"github.com/gin-gonic/gin"
)

/*
 * newTestRouter mounts a protected route behind the real middleware.
 *
 * ExceptionHandler is not optional here. Authenticate no longer writes its own
 * rejection body — it reports the failure with ctx.Error and aborts, so that the
 * status, the envelope and the logging all happen in one place. Without the
 * handler registered, every rejection below would abort with no body at all.
 */
func newTestRouter(t *testing.T) (*gin.Engine, *services.AuthService) {
	t.Helper()

	auth := support.AuthService(t, support.TestSecret)

	gin.SetMode(gin.TestMode)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	router := gin.New()
	router.Use(gin.Recovery(), middlewares.ExceptionHandler(logger))
	router.GET("/protected", middlewares.Authenticate(auth), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"userId":   ctx.GetInt64("userId"),
			"username": ctx.GetString("username"),
		})
	})

	return router, auth
}

func TestProtectedRouteRejectsMalformedHeaders(t *testing.T) {
	router, _ := newTestRouter(t)

	tests := []struct {
		name   string
		header string
	}{
		// Regression: slicing by len("Bearer ") panicked on any shorter header.
		{name: "shorter than the Bearer prefix", header: "abc"},
		{name: "prefix only", header: middlewares.BearerPrefix},
		{name: "prefix with no space", header: "Bearer"},
		{name: "wrong scheme", header: "Basic dXNlcjpwYXNz"},
		{name: "missing", header: ""},
		{name: "bare token without scheme", header: "not-a-jwt"},
		{name: "well-formed scheme, garbage token", header: middlewares.BearerPrefix + "not-a-jwt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := get(t, router, tt.header); res.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d (body: %s)", res.Code, http.StatusUnauthorized, res.Body)
			}
		})
	}
}

// A service holding a different key stands in for an attacker. The router keeps
// the config it was built with, which is the point of injecting it.
func TestProtectedRouteRejectsForeignlySignedToken(t *testing.T) {
	router, _ := newTestRouter(t)

	forged, err := support.AuthService(t, support.OtherSecret).GenerateToken(1, "attacker")
	if err != nil {
		t.Fatalf("GenerateToken() = %v, want nil", err)
	}

	if res := get(t, router, middlewares.BearerPrefix+forged); res.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (body: %s)", res.Code, http.StatusUnauthorized, res.Body)
	}
}

/*
 * A rejected request must not leak parser internals back to the caller.
 *
 * Asserted byte for byte, including the null data and errors keys: the envelope
 * always carries all four, so a client can read response.errors without first
 * checking that the key exists. The cause — expired, forged, malformed — is
 * recorded on the exception for the log and appears nowhere here.
 */
func TestProtectedRouteDoesNotLeakParseErrors(t *testing.T) {
	router, _ := newTestRouter(t)

	res := get(t, router, middlewares.BearerPrefix+"not-a-jwt")

	const want = `{"status":"error","message":"Unauthorized","data":null,"errors":null}`

	if body := res.Body.String(); body != want {
		t.Errorf("body = %s, want %s", body, want)
	}
}

func TestProtectedRouteAcceptsValidToken(t *testing.T) {
	router, auth := newTestRouter(t)

	token, err := auth.GenerateToken(42, "islacks")
	if err != nil {
		t.Fatalf("GenerateToken() = %v, want nil", err)
	}

	res := get(t, router, middlewares.BearerPrefix+token)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", res.Code, http.StatusOK, res.Body)
	}

	if body := res.Body.String(); body != `{"userId":42,"username":"islacks"}` {
		t.Errorf("body = %s, want the claims echoed back", body)
	}
}

// get issues a GET against the throwaway /protected route above. It is local to
// this file on purpose: /protected exists only for these middleware tests.
func get(t *testing.T, router *gin.Engine, authorization string) *httptest.ResponseRecorder {
	t.Helper()

	return support.GetPath(t, router, "/protected", authorization)
}
