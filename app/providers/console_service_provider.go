package providers

import (
	"log"

	consoleRoute "web-app/routes/console"
)

type ConsoleServiceProvider struct {
	cmdName string
	cmdArgs []string
}

// NewConsoleServiceProvider creates a new console service provider
func NewConsoleServiceProvider(cmdName string, cmdArgs []string) *ConsoleServiceProvider {
	return &ConsoleServiceProvider{
		cmdName: cmdName,
		cmdArgs: cmdArgs,
	}
}

// Boot starts the console service provider
func (provider *ConsoleServiceProvider) Boot() {
	// Register commands route
	commands := consoleRoute.Register()
	cmd, ok := commands[provider.cmdName]
	if !ok {
		log.Fatalf("Command %s not found", provider.cmdName)
	}

	// Execute the command
	if err := cmd.Handle(provider.cmdArgs); err != nil {
		log.Fatalf("Error executing command %s: %v", provider.cmdName, err)
	}
}
