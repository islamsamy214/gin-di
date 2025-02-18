package providers

import (
	"log"

	"github.com/joho/godotenv"
)

type EnvServiceProvider struct{}

func NewEnvServiceProvider() *EnvServiceProvider {
	return &EnvServiceProvider{}
}

func (e *EnvServiceProvider) Boot() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}
}
