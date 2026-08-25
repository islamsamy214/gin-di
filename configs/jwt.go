package configs

import (
	"errors"
	"fmt"
	"time"
	"web-app/app/helpers"
)

// MinSecretKeyLength is the HMAC-SHA256 block size; shorter keys are
// zero-padded by the HMAC construction and add no entropy. It is a floor on
// what this config will accept, not a tunable setting.
const MinSecretKeyLength = 32

var (
	ErrMissingJwtSecret = errors.New("JWT_SECRET is not set")
	ErrEmptyJwtIssuer   = errors.New("JWT_ISSUER must not be empty")
)

type JwtConfig struct {
	SecretKey []byte
	Issuer    string
	TTL       time.Duration
}

/**
 * Initialize the JWT configuration.
 *
 * This function loads the JWT configuration from the environment.
 * If the configuration is not set, it defaults to sensible values.
 *
 * The environment is read here rather than in package-level variables, which
 * would be evaluated before main() calls godotenv.Load() and so would never
 * see the values in .env.
 *
 * Unlike the other configs this one also returns an error: the signing key has
 * no safe default, and quietly falling back would make every issued token
 * forgeable.
 *
 * @return *JwtConfig The JWT configuration instance.
 * @return error      If JWT_SECRET, JWT_ISSUER or JWT_TTL is invalid.
 */
func NewJwtConfig() (*JwtConfig, error) {
	config := &JwtConfig{
		/**
		 * The secret key used to sign JWT tokens.
		 * Required: there is no default, an unset key is an error.
		 */
		SecretKey: []byte(helpers.Env("JWT_SECRET", "").(string)),

		/**
		 * The issuer stamped into, and required of, every token.
		 * Defaults to "github@islamsamy214" if JWT_ISSUER is not set.
		 */
		Issuer: helpers.Env("JWT_ISSUER", "github@islamsamy214").(string),

		/**
		 * How long an issued token stays valid, in seconds.
		 * Defaults to 86400 (24 hours) if JWT_TTL is not set.
		 */
		TTL: time.Duration(helpers.Env("JWT_TTL", 86400).(int)) * time.Second,
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

/**
 * Validate the resolved configuration.
 *
 * Rejects values that would issue forgeable tokens, skip issuer verification,
 * or expire on arrival.
 *
 * @return error The first problem found, or nil.
 */
func (config *JwtConfig) validate() error {
	if len(config.SecretKey) == 0 {
		return ErrMissingJwtSecret
	}

	if len(config.SecretKey) < MinSecretKeyLength {
		return fmt.Errorf("JWT_SECRET must be at least %d bytes, got %d", MinSecretKeyLength, len(config.SecretKey))
	}

	if config.Issuer == "" {
		return ErrEmptyJwtIssuer
	}

	if config.TTL <= 0 {
		return fmt.Errorf("JWT_TTL must be a positive number of seconds, got %s", config.TTL)
	}

	return nil
}
