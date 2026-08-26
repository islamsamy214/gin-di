package core

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"
	"web-app/configs"

	_ "github.com/lib/pq"
)

// Connection lifetimes. Neither was set before, so a pooled connection was
// reused indefinitely and went stale behind any proxy that ages idle sockets
// out from under the client.
const (
	connMaxLifetime = 30 * time.Minute
	connMaxIdleTime = 5 * time.Minute
)

// pingTimeout bounds the one eager round trip made at open time.
const pingTimeout = 5 * time.Second

/*
 * connection memoizes the process-wide pool.
 *
 * This mirrors Laravel's DatabaseManager being a container singleton: a model
 * resolves a connection, it does not open one. Before this, every
 * NewUserModel()/NewEventModel() call ran sql.Open and leaked a fresh pool that
 * was never closed — one per request — which exhausts the server's connection
 * limit under any real load.
 *
 * The error is memoized alongside the value, so a failed open is reported to
 * every caller instead of being retried per request against a database that is
 * known to be down.
 */
var connection = sync.OnceValues(NewPostgresService)

type PostgresService struct {
	db *sql.DB
}

/*
 * Connection returns the shared pool, opening it on first use.
 *
 * This is what application code should call. NewPostgresService remains
 * exported for the rare caller that genuinely needs an independent pool it will
 * close itself.
 *
 * @return *PostgresService The shared connection.
 * @return error            If the pool could not be opened or reached.
 */
func Connection() (*PostgresService, error) {
	return connection()
}

/*
 * NewPostgresService opens a new connection pool.
 *
 * Prefer Connection: an unclosed pool returned from here is a resource leak.
 *
 * @return *PostgresService The connection.
 * @return error            If the driver is unsupported, or the pool could not
 *                          be opened or reached.
 */
func NewPostgresService() (*PostgresService, error) {
	databaseConfig := configs.NewDatabaseConfig()

	// Fail fast on a driver this DSN builder cannot speak. The value used to be
	// read and then ignored, so a mysql configuration silently dialled Postgres.
	if databaseConfig.Connection != configs.PostgresConnection {
		return nil, fmt.Errorf(
			"unsupported DB_CONNECTION %q: this application implements %q only",
			databaseConfig.Connection, configs.PostgresConnection,
		)
	}

	db, err := sql.Open("postgres", dataSourceName(databaseConfig))
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	db.SetMaxOpenConns(databaseConfig.MaxOpenConns)
	db.SetMaxIdleConns(databaseConfig.MaxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	// sql.Open is lazy and validates nothing, so without this the first failure
	// surfaces inside an unrelated request rather than at startup.
	pingCtx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		// The half-open pool must not outlive the failed constructor.
		_ = db.Close()

		return nil, fmt.Errorf("reaching database at %s: %w", net.JoinHostPort(databaseConfig.Host, databaseConfig.Port), err)
	}

	return &PostgresService{db: db}, nil
}

/*
 * dataSourceName builds the connection URL.
 *
 * A URL rather than concatenated key=value pairs: the password was previously
 * interpolated raw, so a password containing a space truncated the DSN and the
 * process connected to a different database than the one configured. url.URL
 * escapes every component, and JoinHostPort keeps an IPv6 literal valid.
 *
 * @param config The resolved database configuration.
 * @return string The postgres:// DSN.
 */
func dataSourceName(config *configs.DatabaseConfig) string {
	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.Username, config.Password),
		Host:   net.JoinHostPort(config.Host, config.Port),
		Path:   config.Database,
	}

	dsn.RawQuery = url.Values{"sslmode": {config.SSLMode}}.Encode()

	return dsn.String()
}

// Close closes the database connection.
func (s *PostgresService) Close() error {
	return s.db.Close()
}

// Create runs an INSERT query with a RETURNING clause.
func (s *PostgresService) Create(query string, args ...any) (*sql.Row, error) {
	return s.db.QueryRow(query, args...), nil
}

/*
 * Read runs a SELECT query.
 *
 * QueryRow/Query rather than Prepare-Query-Close: preparing a statement that is
 * executed once costs two extra round trips for nothing, and the driver already
 * handles parameter binding.
 */
func (s *PostgresService) Read(query string, args ...any) (*sql.Rows, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}

	return rows, nil
}

// Update runs an UPDATE query.
func (s *PostgresService) Update(query string, args ...any) (*sql.Row, error) {
	return s.db.QueryRow(query, args...), nil
}

// Delete runs a DELETE query.
func (s *PostgresService) Delete(query string, args ...any) (sql.Result, error) {
	result, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("executing statement: %w", err)
	}

	return result, nil
}

// Begin starts a transaction.
func (s *PostgresService) Begin() (*sql.Tx, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}

	return tx, nil
}
