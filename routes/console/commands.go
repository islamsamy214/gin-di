// route/interfaces/commands.go
package interfaces

import (
	"web-app/app/console"
	"web-app/app/interfaces"
)

func Register() map[string]interfaces.Command {
	// Register the command
	return map[string]interfaces.Command{
		"example": console.NewExampleCommand(),
		"migrate": console.NewMigrateCommand(),
	}
}
