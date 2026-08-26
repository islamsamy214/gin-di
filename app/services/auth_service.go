package services

import (
	"fmt"
	"time"
	"web-app/configs"

	"github.com/golang-jwt/jwt/v5"
)

// AuthService issues and verifies access tokens. Its JWT settings are injected
// rather than resolved here, so nothing in this package reads the environment.
type AuthService struct {
	jwt *configs.JwtConfig
}

func NewAuthService(jwtConfig *configs.JwtConfig) *AuthService {
	return &AuthService{jwt: jwtConfig}
}

// Claims represents the JWT claims structure
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token with the provided user ID
func (s *AuthService) GenerateToken(userID int64, username string) (string, error) {
	now := time.Now()

	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwt.TTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    s.jwt.Issuer,
		},
	}

	signedToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwt.SecretKey)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return signedToken, nil
}

// ParseToken parses the provided JWT token string and returns the claims if the token is valid
func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	// WithValidMethods pins HS256 so a token cannot dictate its own algorithm.
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{},
		func(*jwt.Token) (any, error) { return s.jwt.SecretKey, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.jwt.Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

// Password hashing moved to app/services/hash (the Hash facade) and credential
// verification to UserService. AttemptLogin took a *models.User and then
// rebound that same parameter from the database, and was called as
// AttemptLogin(user, user.Password) — a signature that could not be read
// correctly. UserService.Authenticate(username, password) replaces it.
