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
	CORSOrigins    []string
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
		 * The public base URL this API is reached at, scheme and host included.
		 *
		 * Deliberately has no default. Its only consumer is the CORS allowlist,
		 * where it supplies this application's own origin, and a guessed value
		 * is wrong in every deployment that matters: a default without a port
		 * does not match a browser on http://localhost:8000, because a port is
		 * part of an origin, and no default can know the public host behind a
		 * proxy. Wrong here means same-origin writes are refused with a bare 403
		 * that mentions neither CORS nor the origin, so an empty value is left to
		 * fail loudly at boot instead — and only when CORS is actually enabled.
		 */
		URL: helpers.Env("APP_URL", "").(string),

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
		 * nothing at all when there isn't one. Entries are addresses or CIDRs,
		 * never URLs: "10.0.0.7", "172.16.0.0/12". A URL here is a boot error.
		 */
		TrustedProxies: helpers.EnvSlice("APP_TRUSTED_PROXIES", []string{}),

		/**
		 * The browser origins allowed to call this API cross-origin.
		 *
		 * These are the origins of the pages calling this API, not where the API
		 * itself lives — a browser only sends an Origin header when the two
		 * differ, so listing this API's public address here buys nothing for
		 * cross-origin traffic.
		 *
		 * URL is nevertheless merged in automatically whenever this list is
		 * non-empty, for a narrower reason: a browser also sends Origin on
		 * same-origin requests that are not GET or HEAD, so a frontend served
		 * from this API's own host would have its POSTs refused with a 403
		 * unless that origin is allowed. See middlewares.EffectiveOrigins.
		 *
		 * Empty by default, and empty means the CORS middleware is not
		 * installed at all: a server-to-server or CLI client never sends
		 * Origin, so a pure API deployment should emit no CORS headers rather
		 * than headers that permit nothing. Entries are scheme://host[:port]
		 * with no trailing slash and no path — exactly what a browser sends —
		 * or the single wildcard "*".
		 */
		CORSOrigins: helpers.EnvSlice("APP_CORS_ORIGINS", []string{}),
	}
}
