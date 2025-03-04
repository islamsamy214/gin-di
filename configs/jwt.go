package configs

import "web-app/app/helpers"

type JwtConfig struct {
	SecretKey string
}

/**
 * Initialize the JWT configuration.
 *
 * This function loads the JWT secret key from the environment.
 * If the key is not set, it defaults
 *
 * @return *JwtConfig The JWT configuration instance.
 */
func NewJwtConfig() *JwtConfig {
	return &JwtConfig{
		/**
		 * The secret key used to sign JWT tokens.
		 * Defaults: "secret"
		 */
		SecretKey: helpers.Env("JWT_SECRET", "secret").(string),
	}
}
