package unit

import (
	"errors"
	"testing"
	"time"
	"web-app/app/services"
	"web-app/configs"
	"web-app/tests/support"

	"github.com/golang-jwt/jwt/v5"
)

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
	auth, cfg := support.AuthServiceWithConfig(t, support.TestSecret)

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
	auth, cfg := support.AuthServiceWithConfig(t, support.TestSecret)

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
			token:   signWith(t, jwt.SigningMethodHS256, []byte(support.OtherSecret), validClaims(cfg)),
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
	auth, cfg := support.AuthServiceWithConfig(t, support.TestSecret)

	tokenStr := signWith(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims(cfg))

	claims, err := auth.ParseToken(tokenStr)
	if claims != nil {
		t.Errorf("ParseToken() claims = %+v, want nil", claims)
	}

	if !errors.Is(err, jwt.ErrTokenSignatureInvalid) {
		t.Errorf("ParseToken() = %v, want %v", err, jwt.ErrTokenSignatureInvalid)
	}
}
