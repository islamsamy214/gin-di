package support

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"web-app/app/services"
	httpApis "web-app/routes/http"

	"github.com/gin-gonic/gin"
)

/*
 * AppRouter mounts the real route table, so tests exercise the same wiring
 * main() builds rather than a stand-in.
 *
 * @return *gin.Engine            The router.
 * @return *services.AuthService  The service its middleware was built with.
 */
func AppRouter(t *testing.T) (*gin.Engine, *services.AuthService) {
	t.Helper()

	auth := AuthService(t, TestSecret)

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(gin.Recovery())
	httpApis.Register(router, auth)

	return router, auth
}

// V1Path prefixes a route with the v1 API mount point, so a prefix change stays
// a one-line edit rather than a sweep through every suite.
func V1Path(route string) string {
	return httpApis.APIPrefix + "/v1" + route
}

// PostJSON sends a JSON body to the given path.
func PostJSON(t *testing.T, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}

// GetPath issues a GET, optionally carrying an Authorization header.
func GetPath(t *testing.T, router *gin.Engine, path, authorization string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, path, nil)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	return response
}
