package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"time"
	"web-app/app/models/user"
	"web-app/configs"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/argon2"
)

var (
	// Define a secret key for signing tokens. This should be securely stored in a real-world application.
	secretKey = []byte(configs.NewJwtConfig().SecretKey)
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.StandardClaims
}

// GenerateToken generates a new JWT token with the provided user ID
func GenerateToken(userID int64, username string) (string, error) {
	// Create a new set of claims
	claims := &Claims{
		UserID:   userID,
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(), // Token expiration time
			Issuer:    "github@islamsamy214",
		},
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with our secret key
	signedToken, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// ParseToken parses the provided JWT token string and returns the claims if the token is valid
func ParseToken(tokenStr string) (*Claims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}

	// Check if the token is valid
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, jwt.ErrSignatureInvalid
	}
}

// ValidateToken validates the provided JWT token string
func ValidateToken(tokenStr string) error {
	// Parse the token
	_, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return err
	}

	return nil
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

func AttemptLogin(user *user.User, password string) (*user.User, error) {
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

func GetUserByUsername(u *user.User) (*user.User, error) {
	// Find the user by username
	err := u.FindByUsername()
	if err != nil {
		return nil, err
	}

	return u, nil
}
