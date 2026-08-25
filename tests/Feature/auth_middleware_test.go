package feature

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"web-app/app/http/middlewares"
	"web-app/app/services"
	"web-app/configs"

	"github.com/gin-gonic/gin"
)

const (
	testSecret  = "feature-test-secret-key-of-32-plus-bytes"
	otherSecret = "another-feature-secret-key-of-32-plus-bytes"
)

// authService resolves a real config from the environment and builds the
// service from it, the same way the HTTP provider does at boot.
func authService(t *testing.T, secret string) *services.AuthService {
	t.Helper()

	t.Setenv("JWT_SECRET", secret)

	cfg, err := configs.NewJwtConfig()
	if err != nil {
		t.Fatalf("NewJwtConfig() = %v, want nil", err)
	}

	return services.NewAuthService(cfg)
}

// newTestRouter mounts a protected route behind the real middleware, so these
// tests exercise the same gin.Recovery stack production runs.
func newTestRouter(t *testing.T) (*gin.Engine, *services.AuthService) {
	t.Helper()

	auth := authService(t, testSecret)

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/protected", middlewares.Authenticate(auth), func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"userId":   ctx.GetInt64("userId"),
			"username": ctx.GetString("username"),
		})
	})

	return router, auth
}

func get(t *testing.T, router *gin.Engine, authorization string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	return res
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

	forged, err := authService(t, otherSecret).GenerateToken(1, "attacker")
	if err != nil {
		t.Fatalf("GenerateToken() = %v, want nil", err)
	}

	if res := get(t, router, middlewares.BearerPrefix+forged); res.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (body: %s)", res.Code, http.StatusUnauthorized, res.Body)
	}
}

// A rejected request must not leak parser internals back to the caller.
func TestProtectedRouteDoesNotLeakParseErrors(t *testing.T) {
	router, _ := newTestRouter(t)

	res := get(t, router, middlewares.BearerPrefix+"not-a-jwt")

	if body := res.Body.String(); body != `{"message":"Unauthorized"}` {
		t.Errorf("body = %s, want a fixed Unauthorized message", body)
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
