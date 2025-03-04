package configs

import "web-app/app/helpers"

type DatabaseConfig struct {
	Connection string
	Host       string
	Port       string
	Database   string
	Username   string
	Password   string
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
		 * Defaults to "mysql" if DB_DRIVER is not set.
		 */
		Connection: helpers.Env("DB_DRIVER", "mysql").(string),

		/**
		 * The database host.
		 * Defaults to "127.0.0.1" if DB_HOST is not set.
		 */
		Host: helpers.Env("DB_HOST", "127.0.0.1").(string),

		/**
		 * The database port.
		 * Defaults to "3306" if DB_PORT is not set.
		 */
		Port: helpers.Env("DB_PORT", "3306").(string),

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
	}
}
