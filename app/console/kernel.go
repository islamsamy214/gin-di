package main

import (
	"fmt"
	"os"
	"web-app/app/console/commands"
)

func main() {
	// List all the arguments that are passed to the console
	args := os.Args

	// Ensure at least one argument is provided
	if len(args) < 2 {
		fmt.Println("Usage: go run app/console/kernel.go <command>")
		os.Exit(1)
	}

	// Check if the first argument is "migrate"
	if args[1] == "migrate" {
		commands.Migrate()
	} else {
		fmt.Printf("Unknown command: %s\n", args[1])
		os.Exit(1)
	}
}
