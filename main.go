package main

import (
	"log"
	"os"
	"web-app/app/providers"
)

func main() {
	// Start server on http
	if len(os.Args) > 1 && os.Args[1] == "http" {
		providers.NewHttpServiceProvider().Boot()
	}

	// Any other commands should go as console commands
	if len(os.Args) > 1 {
		providers.NewConsoleServiceProvider().Boot()
	}

	log.Println("You need to pass a command to run the application")
}
