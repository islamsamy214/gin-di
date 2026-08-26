package configs

import "web-app/app/helpers"

type AppConfig struct {
	Name           string
	Env            string
	Debug          bool
	URL            string
	Host           string
	Port           string
	LogLevel       string
	TrustedProxies []string
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
		URL: helpers.Env("APP_URL", "http://localhost").(string),

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

		/**
		 * The minimum severity written to the log.
		 * One of "debug", "info", "warn", "error". Defaults to "info".
		 */
		LogLevel: helpers.Env("APP_LOG_LEVEL", "info").(string),

		/**
		 * The proxies whose forwarding headers are believed when resolving the
		 * client IP, as addresses or CIDRs.
		 *
		 * Empty by default, and empty means trust nothing. Gin trusts every
		 * proxy unless told otherwise and its own documentation calls that
		 * "NOT safe": with no list configured, any caller can dictate the
		 * address that lands in the access log by sending X-Forwarded-For.
		 * Set this to the reverse proxy actually in front of the app, and to
		 * nothing at all when there isn't one.
		 * 172.16.0.0/12 — or better, the container's /32, nginx/Traefik in the same compose network
		 */
		TrustedProxies: helpers.EnvSlice("APP_TRUSTED_PROXIES", []string{}),
	}
}
