package main

import (
	"fmt"
	"os"
	"web-app/app/console/commands"
	"web-app/app/providers"
)

/*
 * The main function is the entry point of the cli application
 */
func main() {
	// List all the arguments that are passed to the console
	args := os.Args

	// Ensure at least one argument is provided
	if len(args) < 2 {
		fmt.Println("Usage: go run app/console/kernel.go <command>")
		os.Exit(1)
	}

	// Load the .env file
	(&providers.EnvServiceProvider{}).Boot()

	// Gin's default logger to use our custom writer
	(&providers.LogServiceProvider{}).Boot()

	// Check if the first argument is "migrate"
	if args[1] == "migrate" {
		// get the flags --down
		if len(args) > 2 && args[2] == "--down" {
			commands.Rollback(args)
			return
		}

		commands.Migrate()
	}

	// Check if the first argument is "seed"
	if args[1] == "seed" {
		commands.Seed()
	}
}
