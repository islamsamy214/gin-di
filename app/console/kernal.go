package main

import (
	"fmt"
	"os"
)

func main() {
	// List all the arguments that are passed to the console
	args := os.Args

	// Ensure at least one argument is provided
	if len(args) < 2 {
		fmt.Println("Usage: go run app/console/kernal.go <command>")
		os.Exit(1)
	}

	// Check if the first argument is "list-routes"
	// if args[1] == "list-routes" {
	// 	commands.ListRoutes()
	// } else {
	// 	fmt.Printf("Unknown command: %s\n", args[1])
	// 	os.Exit(1)
	// }
}
