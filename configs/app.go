package configs

import "web-app/app/helpers"

type AppConfig struct {
	Name  string
	Env   string
	Debug bool
	Url   string
	Host  string
	Port  string
}

/**
 * Initialize the application configuration.
 *
 * This function loads environment variables and assigns them to the AppConfig struct.
 * It follows Laravel's configuration style, where values are retrieved from the environment
 * with sensible defaults to ensure stability.
 *
 * @return *AppConfig The application configuration instance.
 */
func NewAppConfig() *AppConfig {
	return &AppConfig{
		/**
		 * The application name.
		 * Defaults to an empty string if APP_NAME is not set.
		 */
		Name: helpers.Env("APP_NAME", "").(string),

		/**
		 * The application environment (e.g., "local", "production", "staging").
		 * Defaults to "production" if APP_ENV is not set.
		 */
		Env: helpers.Env("APP_ENV", "production").(string),

		/**
		 * Application debug mode.
		 * If true, detailed error messages will be displayed.
		 * Defaults to false if APP_DEBUG is not set.
		 */
		Debug: helpers.Env("APP_DEBUG", false).(bool),

		/**
		 * The base URL of the application.
		 * Defaults to "http://localhost" if APP_URL is not set.
		 */
		Url: helpers.Env("APP_URL", "http://localhost").(string),

		/**
		 * The host the application runs on.
		 * Defaults to "127.0.0.1" if APP_HOST is not set.
		 */
		Host: helpers.Env("APP_HOST", "127.0.0.1").(string),

		/**
		 * The port number the application is listening on.
		 * Defaults to 8000 if APP_PORT is not set.
		 */
		Port: helpers.Env("APP_PORT", "8000").(string),
	}
}
