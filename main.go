package main

import (
	"log"
	"os"
	"web-app/app/providers"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	// Start server on http
	if len(os.Args) > 1 && os.Args[1] == "http" {
		providers.NewHttpServiceProvider().Boot()
	}

	// Any other commands should go as console commands
	if len(os.Args) >= 2 {
		providers.NewConsoleServiceProvider(os.Args[1], os.Args[2:]).Boot()
		os.Exit(0)
	}

	log.Println("Usage: go run main.go <command>")
	os.Exit(1)
}
