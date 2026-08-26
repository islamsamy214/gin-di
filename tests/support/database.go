package support

import (
	"os"
	"testing"
	"web-app/app/console/test"
	"web-app/app/services/core"
	"web-app/database"
)

// DatabaseTestsEnv must be set for the database tests to run. These tests DROP
// EVERY TABLE, so they stay opt-in rather than risking a developer's working
// database on a stray `go test ./...`.
const DatabaseTestsEnv = test.DatabaseTestsEnv

/*
 * FreshDatabase gives the test an empty, fully migrated schema.
 *
 * Equivalent to Laravel's RefreshDatabase trait. Skips rather than fails when
 * the opt-in is absent or no database answers, so the suite stays green on a
 * machine without Postgres.
 *
 * @return *core.PostgresService The shared connection. Deliberately NOT closed on
 *                               cleanup — see below.
 */
func FreshDatabase(t *testing.T) *core.PostgresService {
	t.Helper()

	if os.Getenv(DatabaseTestsEnv) == "" {
		t.Skipf("set %s=1 to run database tests (they drop every table)", DatabaseTestsEnv)
	}

	/*
	 * The shared, memoized pool — the same one the models resolve, so a test
	 * exercises the connection the application actually uses.
	 *
	 * There is deliberately no t.Cleanup closing it. The pool is process-wide, so
	 * closing it after the first test would leave every later test in the same
	 * binary holding a closed handle. It is released when the process exits.
	 */
	db, err := core.Connection()
	if err != nil {
		t.Skipf("no database reachable: %v", err)
	}

	// Connection pings at open time, so reaching here means Postgres answered.
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
