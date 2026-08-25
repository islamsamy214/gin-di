package feature

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"web-app/app/console"
	"web-app/app/services"
	"web-app/app/services/core"
	"web-app/database"
	httpApis "web-app/routes/http"

	"github.com/gin-gonic/gin"
)

// runDatabaseTestsEnv must be set for the database tests to run. These tests
// DROP EVERY TABLE, so they stay opt-in rather than risking a developer's
// working database on a stray `go test ./...`.
const runDatabaseTestsEnv = console.DatabaseTestsEnv

/*
 * freshDatabase gives the test an empty, fully migrated schema.
 *
 * Equivalent to Laravel's RefreshDatabase trait. Skips rather than fails when
 * the opt-in is absent or no database answers, so the suite stays green on a
 * machine without Postgres.
 *
 * @return *core.PostgresService The live connection, closed on cleanup.
 */
func freshDatabase(t *testing.T) *core.PostgresService {
	t.Helper()

	if os.Getenv(runDatabaseTestsEnv) == "" {
		t.Skipf("set %s=1 to run database tests (they drop every table)", runDatabaseTestsEnv)
	}

	db, err := core.NewPostgresService()
	if err != nil {
		t.Skipf("cannot open a database connection: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing the database: %v", err)
		}
	})

	// sql.Open is lazy, so this is the first attempt to actually reach Postgres.
	rows, err := db.Read(`SELECT 1`)
	if err != nil {
		t.Skipf("no database reachable: %v", err)
	}

	if err := rows.Close(); err != nil {
		t.Fatalf("closing the probe rows: %v", err)
	}

	if err := database.NewMigrator(db, database.NewKernel().Migrations).Fresh(); err != nil {
		t.Fatalf("preparing a fresh schema: %v", err)
	}

	return db
}

/*
 * appRouter mounts the real route table, so these tests exercise the same
 * wiring main() builds rather than a stand-in.
 *
 * @return *gin.Engine            The router.
 * @return *services.AuthService  The service its middleware was built with.
 */
func appRouter(t *testing.T) (*gin.Engine, *services.AuthService) {
	t.Helper()

	auth := authService(t, testSecret)

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(gin.Recovery())
	httpApis.Regester(router, auth)

	return router, auth
}

// postJSON sends a JSON body to the given path.
func postJSON(t *testing.T, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	return res
}

func get(t *testing.T, router *gin.Engine, authorization string) *httptest.ResponseRecorder {
	t.Helper()

	return getPath(t, router, "/protected", authorization)
}

func getPath(t *testing.T, router *gin.Engine, path, authorization string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	return res
}
