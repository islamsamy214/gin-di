package configs

import "web-app/app/helpers"

// PostgresConnection is the only driver this application implements. It is the
// value DB_CONNECTION must carry; anything else is a configuration error rather
// than a silent fallback, because the DSN builder speaks Postgres only.
const PostgresConnection = "pgsql"

// Connection pool defaults. Both are per-process, so the ceiling a database
// sees is these multiplied by the number of replicas.
const (
	defaultMaxOpenConns = 10
	defaultMaxIdleConns = 5
)

type DatabaseConfig struct {
	Connection   string
	Host         string
	Port         string
	Database     string
	Username     string
	Password     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

/**
 * Initialize the database configuration.
 *
 * This function loads the database configuration from the environment.
 * If the configuration is not set, it defaults to sensible values.
 *
 * @return *DatabaseConfig The database configuration instance.
 */
func NewDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		/**
		 * The database driver.
		 * Defaults to "pgsql" if DB_CONNECTION is not set.
		 *
		 * The key was DB_DRIVER while .env.example set DB_CONNECTION, so it
		 * never matched and the old "mysql" default always won — harmless only
		 * because the value was then discarded by the DSN builder. Both halves
		 * are fixed: the key matches, and the value is now checked.
		 */
		Connection: helpers.Env("DB_CONNECTION", PostgresConnection).(string),

		/**
		 * The database host.
		 * Defaults to "127.0.0.1" if DB_HOST is not set.
		 */
		Host: helpers.Env("DB_HOST", "127.0.0.1").(string),

		/**
		 * The database port.
		 * Defaults to "5432" if DB_PORT is not set.
		 */
		Port: helpers.Env("DB_PORT", "5432").(string),

		/**
		 * The database name.
		 * Defaults to an empty string if DB_DATABASE is not set.
		 */
		Database: helpers.Env("DB_DATABASE", "homestead").(string),

		/**
		 * The database username.
		 * Defaults to "root" if DB_USERNAME is not set.
		 */
		Username: helpers.Env("DB_USERNAME", "root").(string),

		/**
		 * The database password.
		 * Defaults to an empty string if DB_PASSWORD is not set.
		 */
		Password: helpers.Env("DB_PASSWORD", "").(string),

		/**
		 * How TLS is negotiated with the server.
		 *
		 * Defaults to "prefer": encrypt when the server offers it, rather than
		 * the "disable" that used to be hardcoded into the DSN with no way to
		 * override it. Production should set "require" at minimum, and
		 * "verify-full" where the server certificate can be verified.
		 */
		SSLMode: helpers.Env("DB_SSLMODE", "prefer").(string),

		/**
		 * The maximum number of open connections this process may hold.
		 */
		MaxOpenConns: helpers.Env("DB_MAX_OPEN_CONNS", defaultMaxOpenConns).(int),

		/**
		 * The number of idle connections kept ready in the pool.
		 */
		MaxIdleConns: helpers.Env("DB_MAX_IDLE_CONNS", defaultMaxIdleConns).(int),
	}
}
