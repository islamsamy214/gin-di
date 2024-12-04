package services

import (
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	// Define a secret key for signing tokens. This should be securely stored in a real-world application.
	secretKey = []byte(os.Getenv("JWT_SECRET"))
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.StandardClaims
}

// GenerateToken generates a new JWT token with the provided user ID
func GenerateToken(userID int64) (string, error) {
	// Create a new set of claims
	claims := &Claims{
		UserID: userID,
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
