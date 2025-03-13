package console

import "fmt"

// ExampleCommand is a sample command implementation
type ExampleCommand struct{}

// NewExampleCommand creates a new instance of ExampleCommand
func NewExampleCommand() *ExampleCommand {
	return &ExampleCommand{}
}

// Handle implements the command execution
func (command *ExampleCommand) Handle(args []string) error {
	fmt.Println("Executing example command with args:", args)
	return nil
}

// Description provides information about the command
func (command *ExampleCommand) Description() string {
	return "An example command that prints its arguments"
}
