package configs

import (
	"fmt"
	"time"
	"web-app/app/helpers"
	"web-app/app/services/throttle"
)

/*
 * MemoryStoreDriver is the only rate limit store this application implements.
 *
 * It counts within one process, so the ceiling a client actually meets is this
 * multiplied by the number of replicas. That is the trigger for implementing a
 * shared store rather than a reason to leave the limit off, and it is why the
 * driver is named in configuration at all: adding one is a new implementation of
 * throttle.Store plus a case here, with no route or middleware touched.
 */
const MemoryStoreDriver = "memory"

// Defaults. Sixty per minute is Laravel's api throttle, generous for a person
// and restrictive for a script.
const (
	defaultThrottleLimit      = 60
	defaultThrottleWindow     = 60
	defaultLoginThrottleLimit = 5
)

/*
 * ThrottleConfig is the resolved rate limiting configuration.
 *
 * Global is the allowance every request draws from. Login is separate and much
 * tighter: each attempt on that route costs an argon2id verification, so it is
 * the one endpoint where unlimited retries are expensive as well as insecure.
 */
type ThrottleConfig struct {
	Store   string
	Global  throttle.Limit
	Login   throttle.Limit
	MaxKeys int
}

/*
 * NewThrottleConfig loads the rate limiting configuration.
 *
 * Returns an error like NewJwtConfig, because a misspelled store driver must not
 * degrade into an unthrottled application: the whole point of the setting is to
 * be explicit about where counting happens.
 *
 * @return *ThrottleConfig The configuration.
 * @return error           If the driver is unknown or a value is not usable.
 */
func NewThrottleConfig() (*ThrottleConfig, error) {
	config := &ThrottleConfig{
		/**
		 * Where request counts are kept.
		 * Defaults to MemoryStoreDriver; any other value is an error.
		 */
		Store: helpers.Env("THROTTLE_STORE", MemoryStoreDriver).(string),

		/**
		 * The allowance every request draws from, keyed on client address.
		 *
		 * A limit of zero disables the global throttle: the middleware is not
		 * installed at all, the same way an empty APP_CORS_ORIGINS installs no
		 * CORS. One knob rather than a separate enabled flag, so there is no way
		 * to configure a limit that is silently ignored.
		 */
		Global: throttle.Limit{
			Name:     "global",
			Requests: helpers.Env("THROTTLE_LIMIT", defaultThrottleLimit).(int),
			Per:      time.Duration(helpers.Env("THROTTLE_WINDOW", defaultThrottleWindow).(int)) * time.Second,
		},

		/**
		 * The allowance for POST /login, per minute.
		 *
		 * Its own knob rather than a share of the global one, for two reasons: a
		 * load test needs to relax it without switching off protection
		 * everywhere, and the global tuning knob must not silently disable
		 * brute-force protection on the one route where it matters most.
		 */
		Login: throttle.Limit{
			Name:     "login",
			Requests: helpers.Env("THROTTLE_LOGIN_LIMIT", defaultLoginThrottleLimit).(int),
			Per:      time.Minute,
		},

		/**
		 * How many distinct callers to track before the oldest are evicted.
		 * Defaults to throttle.DefaultMaxKeys.
		 */
		MaxKeys: helpers.Env("THROTTLE_MAX_KEYS", throttle.DefaultMaxKeys).(int),
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

/*
 * GlobalEnabled reports whether the global throttle should be installed.
 *
 * @return bool True when a positive global limit is configured.
 */
func (config *ThrottleConfig) GlobalEnabled() bool {
	return config.Global.Requests > 0
}

/*
 * validate rejects a configuration that would not limit what it claims to.
 *
 * @return error The first problem found, or nil.
 */
func (config *ThrottleConfig) validate() error {
	if config.Store != MemoryStoreDriver {
		return fmt.Errorf(
			"THROTTLE_STORE %q is not implemented; supported drivers: %q",
			config.Store, MemoryStoreDriver,
		)
	}

	// Zero is meaningful — it disables the global throttle — but a negative
	// allowance is a typo that would otherwise refuse every request.
	if config.Global.Requests < 0 {
		return fmt.Errorf("THROTTLE_LIMIT must not be negative, got %d", config.Global.Requests)
	}

	if config.Global.Per <= 0 {
		return fmt.Errorf("THROTTLE_WINDOW must be a positive number of seconds, got %s", config.Global.Per)
	}

	if config.Login.Requests <= 0 {
		return fmt.Errorf("THROTTLE_LOGIN_LIMIT must be positive, got %d", config.Login.Requests)
	}

	if config.MaxKeys <= 0 {
		return fmt.Errorf("THROTTLE_MAX_KEYS must be positive, got %d", config.MaxKeys)
	}

	return nil
}
