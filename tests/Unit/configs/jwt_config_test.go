package unit

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
	"web-app/configs"
	"web-app/tests/support"
)

// unsetEnv makes a variable genuinely absent while still restoring it on
// cleanup: t.Setenv registers the restore, Unsetenv removes the value.
func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		t.Setenv(key, "")

		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unsetting %s: %v", key, err)
		}
	}
}

// defaultConfig resolves the config with every optional variable absent, so
// tests can compare against the real defaults instead of restating them.
func defaultConfig(t *testing.T) *configs.JwtConfig {
	t.Helper()

	t.Setenv("JWT_SECRET", support.TestSecret)
	unsetEnv(t, "JWT_ISSUER", "JWT_TTL")

	cfg, err := configs.NewJwtConfig()
	if err != nil {
		t.Fatalf("NewJwtConfig() = %v, want nil", err)
	}

	return cfg
}

// The default literals live in configs; assert they are usable rather than
// duplicating them here.
func TestNewJwtConfigDefaultsAreUsable(t *testing.T) {
	cfg := defaultConfig(t)

	if cfg.Issuer == "" {
		t.Error("Issuer = empty, want a default issuer")
	}

	if cfg.TTL <= 0 {
		t.Errorf("TTL = %v, want a positive default", cfg.TTL)
	}
}

func TestNewJwtConfigReadsOverrides(t *testing.T) {
	const (
		wantIssuer = "custom-issuer"

		// 900 seconds, so the test fails if the value is taken verbatim as a
		// time.Duration (which would be 900 nanoseconds).
		wantTTL = 15 * time.Minute
	)

	t.Setenv("JWT_SECRET", support.TestSecret)
	t.Setenv("JWT_ISSUER", wantIssuer)
	t.Setenv("JWT_TTL", "900")

	cfg, err := configs.NewJwtConfig()
	if err != nil {
		t.Fatalf("NewJwtConfig() = %v, want nil", err)
	}

	if cfg.Issuer != wantIssuer {
		t.Errorf("Issuer = %q, want %q", cfg.Issuer, wantIssuer)
	}

	if cfg.TTL != wantTTL {
		t.Errorf("TTL = %v, want %v", cfg.TTL, wantTTL)
	}
}

// An explicitly blank issuer would disable issuer verification, so it is an
// error rather than a silent fallback to the default.
func TestNewJwtConfigRejectsEmptyIssuer(t *testing.T) {
	t.Setenv("JWT_SECRET", support.TestSecret)
	t.Setenv("JWT_ISSUER", "")

	cfg, err := configs.NewJwtConfig()
	if cfg != nil {
		t.Errorf("NewJwtConfig() = %+v, want nil", cfg)
	}

	if !errors.Is(err, configs.ErrEmptyJwtIssuer) {
		t.Errorf("NewJwtConfig() = %v, want %v", err, configs.ErrEmptyJwtIssuer)
	}
}

// JWT_TTL is a plain seconds count, so anything non-numeric falls back to the
// default: helpers.Env swallows parse failures for every config value.
// Documented here because it means a typo silently changes token lifetime.
func TestNewJwtConfigFallsBackOnNonNumericTTL(t *testing.T) {
	wantTTL := defaultConfig(t).TTL

	tests := []struct {
		name string
		ttl  string
	}{
		{name: "words", ttl: "soon"},
		{name: "duration string", ttl: "24h"},
		{name: "fractional", ttl: "1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", support.TestSecret)
			t.Setenv("JWT_TTL", tt.ttl)

			cfg, err := configs.NewJwtConfig()
			if err != nil {
				t.Fatalf("NewJwtConfig() = %v, want nil", err)
			}

			if cfg.TTL != wantTTL {
				t.Errorf("TTL = %v, want the default %v", cfg.TTL, wantTTL)
			}
		})
	}
}

// A TTL that parses but cannot issue a usable token is still an error: these
// reach validate() rather than being swallowed by the helper.
func TestNewJwtConfigRejectsNonPositiveTTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  string
	}{
		{name: "zero", ttl: "0"},
		{name: "negative", ttl: "-900"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", support.TestSecret)
			t.Setenv("JWT_TTL", tt.ttl)

			cfg, err := configs.NewJwtConfig()
			if cfg != nil {
				t.Errorf("NewJwtConfig() = %+v, want nil", cfg)
			}

			if err == nil {
				t.Error("NewJwtConfig() = nil error, want an error")
			}
		})
	}
}

func TestNewJwtConfigFailsFastOnSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		unset   bool
		wantErr error
	}{
		{
			name:    "unset",
			unset:   true,
			wantErr: configs.ErrMissingJwtSecret,
		},
		{
			name:    "empty",
			secret:  "",
			wantErr: configs.ErrMissingJwtSecret,
		},
		{
			name:   "one byte short of the minimum",
			secret: strings.Repeat("a", configs.MinSecretKeyLength-1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unset {
				unsetEnv(t, "JWT_SECRET")
			} else {
				t.Setenv("JWT_SECRET", tt.secret)
			}

			cfg, err := configs.NewJwtConfig()
			if cfg != nil {
				t.Errorf("NewJwtConfig() = %+v, want nil", cfg)
			}

			if err == nil {
				t.Fatal("NewJwtConfig() = nil error, want an error")
			}

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("NewJwtConfig() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewJwtConfigAcceptsSecretAtMinimumLength(t *testing.T) {
	secret := strings.Repeat("a", configs.MinSecretKeyLength)

	t.Setenv("JWT_SECRET", secret)

	cfg, err := configs.NewJwtConfig()
	if err != nil {
		t.Fatalf("NewJwtConfig() = %v, want nil", err)
	}

	if string(cfg.SecretKey) != secret {
		t.Errorf("SecretKey = %q, want %q", cfg.SecretKey, secret)
	}
}

// The configured TTL must reach the issued token, proving the service consumes
// the injected config rather than a hardcoded constant.
func TestConfiguredTTLReachesToken(t *testing.T) {
	const wantTTL = 15 * time.Minute

	t.Setenv("JWT_TTL", "900")

	auth, cfg := support.AuthServiceWithConfig(t, support.TestSecret)

	if cfg.TTL != wantTTL {
		t.Fatalf("TTL = %v, want %v", cfg.TTL, wantTTL)
	}

	tokenStr, err := auth.GenerateToken(1, "islacks")
	if err != nil {
		t.Fatalf("GenerateToken() = %v, want nil", err)
	}

	claims, err := auth.ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseToken() = %v, want nil", err)
	}

	if gotTTL := claims.ExpiresAt.Sub(claims.IssuedAt.Time); gotTTL != wantTTL {
		t.Errorf("TTL = %v, want %v", gotTTL, wantTTL)
	}
}

// A token minted under one issuer must not verify under another.
func TestIssuerIsEnforcedAcrossConfigs(t *testing.T) {
	t.Setenv("JWT_ISSUER", "issuer-a")

	authA, _ := support.AuthServiceWithConfig(t, support.TestSecret)

	t.Setenv("JWT_ISSUER", "issuer-b")

	authB, _ := support.AuthServiceWithConfig(t, support.TestSecret)

	tokenStr, err := authA.GenerateToken(1, "islacks")
	if err != nil {
		t.Fatalf("GenerateToken() = %v, want nil", err)
	}

	if _, err := authB.ParseToken(tokenStr); err == nil {
		t.Error("ParseToken() = nil error, want rejection of a foreign issuer")
	}
}

// Guards the package-level fixture against drifting below the enforced minimum.
func TestFixtureSecretMeetsMinimumLength(t *testing.T) {
	for name, secret := range map[string]string{"support.TestSecret": support.TestSecret, "support.OtherSecret": support.OtherSecret} {
		if len(secret) < configs.MinSecretKeyLength {
			t.Errorf("%s is %d bytes, want at least %d", name, len(secret), configs.MinSecretKeyLength)
		}
	}
}
