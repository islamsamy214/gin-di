package interfaces

type Command interface {
	// Handle executes the command with the given arguments
	Handle(args []string) error

	// Description returns the command description
	Description() string
}
