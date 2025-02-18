package configs

import "os"

type AppConfig struct {
	Name  string
	Env   string
	Debug bool
	Url   string
	Host  string
	Port  string
}

func NewApp() *AppConfig {
	// Get app name
	name := os.Getenv("APP_NAME")
	if name == "" {
		name = "Web App"
	}

	// Get app environment
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "production"
	}

	// Get app debug
	debug := false
	if os.Getenv("APP_DEBUG") == "true" {
		debug = true
	}

	// Get app url
	url := os.Getenv("APP_URL")
	if url == "" {
		url = "http://localhost"
	}

	// Get app host
	host := os.Getenv("APP_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	// Get app port
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8000"
	}

	return &AppConfig{
		Name:  name,
		Env:   env,
		Debug: debug,
		Url:   url,
		Host:  host,
		Port:  port,
	}
}
