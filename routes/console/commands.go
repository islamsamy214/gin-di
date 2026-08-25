// route/interfaces/commands.go
package interfaces

import (
	"web-app/app/console"
	"web-app/app/console/jwt"
	"web-app/app/console/migrations"
	"web-app/app/console/test"
	"web-app/app/interfaces"
)

func Register() map[string]interfaces.Command {
	// Register the command
	return map[string]interfaces.Command{
		"example": console.NewExampleCommand(),
		"test":    test.NewCommand(),

		// Database commands
		"migrate":         migrations.NewMigrateCommand(),
		"migrate:fresh":   migrations.NewMigrateFreshCommand(),
		"migrate:refresh": migrations.NewMigrateRefreshCommand(),
		"db:seed":         migrations.NewSeedCommand(),

		// JWT commands
		"jwt:secret": jwt.NewSecretCommand(),
	}
}
