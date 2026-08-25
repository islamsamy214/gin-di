package unit

import (
	"errors"
	"testing"
	"time"
	"web-app/app/services"
	"web-app/configs"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecret  = "unit-test-secret-key-of-at-least-32-bytes"
	otherSecret = "another-unit-secret-key-of-at-least-32-bytes"
)

// jwtConfig resolves a real config from the environment, so the tests exercise
// the same path the HTTP provider and console commands use.
func jwtConfig(t *testing.T, secret string) *configs.JwtConfig {
	t.Helper()

	t.Setenv("JWT_SECRET", secret)

	cfg, err := configs.NewJwtConfig()
	if err != nil {
		t.Fatalf("NewJwtConfig() = %v, want nil", err)
	}

	return cfg
}

func authService(t *testing.T, secret string) (*services.AuthService, *configs.JwtConfig) {
	t.Helper()

	cfg := jwtConfig(t, secret)

	return services.NewAuthService(cfg), cfg
}

// signWith mints a token bypassing GenerateToken, so tests can forge claims and
// signing methods that GenerateToken would never produce.
func signWith(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()

	tokenStr, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("signing test token: %v", err)
	}

	return tokenStr
}

// validClaims returns claims ParseToken accepts, built from the config so the
// expectations follow JWT_ISSUER and JWT_TTL rather than duplicating them.
func validClaims(cfg *configs.JwtConfig) *services.Claims {
	now := time.Now()

	return &services.Claims{
		UserID:   7,
		Username: "islacks",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.TTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    cfg.Issuer,
		},
	}
}

func TestGenerateTokenParseTokenRoundTrip(t *testing.T) {
	auth, cfg := authService(t, testSecret)

	const (
		wantUserID   = int64(42)
		wantUsername = "islacks"
	)

	tokenStr, err := auth.GenerateToken(wantUserID, wantUsername)
	if err != nil {
		t.Fatalf("GenerateToken() = %v, want nil", err)
	}

	claims, err := auth.ParseToken(tokenStr)
	if err != nil {
		t.Fatalf("ParseToken() = %v, want nil", err)
	}

	if claims.UserID != wantUserID {
		t.Errorf("UserID = %d, want %d", claims.UserID, wantUserID)
	}

	if claims.Username != wantUsername {
		t.Errorf("Username = %q, want %q", claims.Username, wantUsername)
	}

	if claims.Issuer != cfg.Issuer {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, cfg.Issuer)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt = nil, want a value")
	}

	if claims.IssuedAt == nil {
		t.Fatal("IssuedAt = nil, want a value")
	}

	// The service must honour the configured TTL, not merely set some expiry.
	if gotTTL := claims.ExpiresAt.Sub(claims.IssuedAt.Time); gotTTL != cfg.TTL {
		t.Errorf("TTL = %v, want %v", gotTTL, cfg.TTL)
	}
}

func TestParseTokenRejectsInvalidTokens(t *testing.T) {
	auth, cfg := authService(t, testSecret)

	expiredClaims := validClaims(cfg)
	expiredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
	expiredClaims.IssuedAt = jwt.NewNumericDate(time.Now().Add(-2 * time.Hour))

	wrongIssuerClaims := validClaims(cfg)
	wrongIssuerClaims.Issuer = "attacker"

	noExpiryClaims := validClaims(cfg)
	noExpiryClaims.ExpiresAt = nil

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "signed with a different key",
			token:   signWith(t, jwt.SigningMethodHS256, []byte(otherSecret), validClaims(cfg)),
			wantErr: jwt.ErrTokenSignatureInvalid,
		},
		{
			name:    "expired",
			token:   signWith(t, jwt.SigningMethodHS256, cfg.SecretKey, expiredClaims),
			wantErr: jwt.ErrTokenExpired,
		},
		{
			name:    "wrong issuer",
			token:   signWith(t, jwt.SigningMethodHS256, cfg.SecretKey, wrongIssuerClaims),
			wantErr: jwt.ErrTokenInvalidIssuer,
		},
		{
			name:    "missing expiry",
			token:   signWith(t, jwt.SigningMethodHS256, cfg.SecretKey, noExpiryClaims),
			wantErr: jwt.ErrTokenRequiredClaimMissing,
		},
		{
			name:    "unexpected signing method HS512",
			token:   signWith(t, jwt.SigningMethodHS512, cfg.SecretKey, validClaims(cfg)),
			wantErr: jwt.ErrTokenSignatureInvalid,
		},
		{
			name:    "garbage",
			token:   "not-a-jwt",
			wantErr: jwt.ErrTokenMalformed,
		},
		{
			name:    "empty",
			token:   "",
			wantErr: jwt.ErrTokenMalformed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := auth.ParseToken(tt.token)
			if claims != nil {
				t.Errorf("ParseToken() claims = %+v, want nil", claims)
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ParseToken() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// The alg:none family is the algorithm-confusion attack the WithValidMethods
// pin exists to stop, so it gets its own test rather than a table row.
func TestParseTokenRejectsAlgNone(t *testing.T) {
	auth, cfg := authService(t, testSecret)

	tokenStr := signWith(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims(cfg))

	claims, err := auth.ParseToken(tokenStr)
	if claims != nil {
		t.Errorf("ParseToken() claims = %+v, want nil", claims)
	}

	if !errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		t.Errorf("ParseToken() = %v, want %v", err, jwt.ErrTokenSignatureInvalid)
	}
}

func TestHashPasswordVerifyPassword(t *testing.T) {
	const password = "correct-horse-battery-staple"

	hash, err := services.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() = %v, want nil", err)
	}

	t.Run("accepts the correct password", func(t *testing.T) {
		match, err := services.VerifyPassword(hash, password)
		if err != nil {
			t.Fatalf("VerifyPassword() = %v, want nil", err)
		}

		if !match {
			t.Error("VerifyPassword() = false, want true")
		}
	})

	t.Run("rejects the wrong password", func(t *testing.T) {
		match, err := services.VerifyPassword(hash, "wrong-password")
		if err != nil {
			t.Fatalf("VerifyPassword() = %v, want nil", err)
		}

		if match {
			t.Error("VerifyPassword() = true, want false")
		}
	})

	t.Run("salt is random per hash", func(t *testing.T) {
		other, err := services.HashPassword(password)
		if err != nil {
			t.Fatalf("HashPassword() = %v, want nil", err)
		}

		if other == hash {
			t.Error("two hashes of the same password are identical, want distinct salts")
		}
	})
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	tests := []struct {
		name string
		hash string
	}{
		{name: "not base64", hash: "!!!not-base64!!!"},
		{name: "shorter than the salt", hash: "AAAA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := services.VerifyPassword(tt.hash, "any-password")
			if err == nil {
				t.Error("VerifyPassword() = nil error, want an error")
			}

			if match {
				t.Error("VerifyPassword() = true, want false")
			}
		})
	}
}
