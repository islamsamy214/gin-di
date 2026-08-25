package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
	"web-app/app/models"
	"web-app/configs"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
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

// AttemptLogin verifies the supplied credentials and returns the matching user
func (s *AuthService) AttemptLogin(user *models.User, password string) (*models.User, error) {
	// Get the user from the database
	user, err := GetUserByUsername(user)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Verify the password
	match, err := VerifyPassword(user.Password, password)
	if err != nil {
		return nil, err
	}

	if !match {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}

// HashPassword hashes the provided password using the Argon2id key derivation function
func HashPassword(password string) (string, error) {
	// Generate a salt with a length of 16 bytes
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash the password using the Argon2id key derivation function
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	fullHash := append(salt, hash...)

	// Encode to a base64 string
	return base64.StdEncoding.EncodeToString(fullHash), nil
}

// decodeHashedPassword decodes the hashed password and returns the password
func VerifyPassword(hashedPassword, password string) (bool, error) {
	// Decode the base64 string to get the full hash (salt + hashed password)
	data, err := base64.StdEncoding.DecodeString(hashedPassword)
	if err != nil {
		return false, err
	}

	// Extract the salt (first 16 bytes)
	if len(data) < 16 {
		return false, errors.New("invalid hash format")
	}
	salt := data[:16]

	// Extract the hash (remaining bytes)
	storedHash := data[16:]

	// Hash the provided password using the same salt
	newHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Compare the new hash with the stored hash
	return subtle.ConstantTimeCompare(newHash, storedHash) == 1, nil
}

func GetUserByUsername(u *models.User) (*models.User, error) {
	// Find the user by username
	err := u.FindByUsername()
	if err != nil {
		return nil, err
	}

	return u, nil
}
