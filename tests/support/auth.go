package support

import (
	"testing"
	"web-app/app/services"
	"web-app/configs"
)

// Secrets shared by every suite. Both must stay at or above
// configs.MinSecretKeyLength; a test asserts exactly that.
const (
	TestSecret  = "shared-test-secret-key-of-32-plus-bytes"
	OtherSecret = "another-shared-secret-key-of-32-plus-bytes"
)

/*
 * JwtConfig resolves a real config from the environment, so tests exercise the
 * same path the HTTP provider and console commands use rather than a stand-in.
 *
 * @param secret The value JWT_SECRET is set to for this test only.
 */
func JwtConfig(t *testing.T, secret string) *configs.JwtConfig {
	t.Helper()

	t.Setenv("JWT_SECRET", secret)

	config, err := configs.NewJwtConfig()
	if err != nil {
		t.Fatalf("NewJwtConfig() = %v, want nil", err)
	}

	return config
}

// AuthService builds the service the way the HTTP provider does at boot.
func AuthService(t *testing.T, secret string) *services.AuthService {
	t.Helper()

	service, _ := AuthServiceWithConfig(t, secret)

	return service
}

/*
 * AuthServiceWithConfig returns the service alongside the very config instance
 * it was built from, for tests that assert on both.
 */
func AuthServiceWithConfig(t *testing.T, secret string) (*services.AuthService, *configs.JwtConfig) {
	t.Helper()

	config := JwtConfig(t, secret)

	return services.NewAuthService(config), config
}
