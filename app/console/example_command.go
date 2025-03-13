package console

import "fmt"

// ExampleCommand is a sample command implementation
type ExampleCommand struct{}

// Handle implements the command execution
func (command *ExampleCommand) Handle(args []string) error {
	fmt.Println("Executing example command with args:", args)
	return nil
}

// Description provides information about the command
func (command *ExampleCommand) Description() string {
	return "An example command that prints its arguments"
}
