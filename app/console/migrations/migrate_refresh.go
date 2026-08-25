package migrations

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"web-app/app/console"
	"web-app/app/services/core"
	"web-app/configs"
	"web-app/database"
)

type MigrateRefreshCommand struct{}

func NewMigrateRefreshCommand() *MigrateRefreshCommand {
	return &MigrateRefreshCommand{}
}

/*
 * Handle rolls every migration back, then runs them again.
 *
 * Unlike migrate:fresh this goes through each migration's Down(), so it also
 * proves the rollback path works. It still destroys the data in those tables,
 * so it asks for confirmation too.
 */
func (command *MigrateRefreshCommand) Handle(args []string) error {
	flags := flag.NewFlagSet("migrate:refresh", flag.ContinueOnError)
	force := flags.Bool("force", false, "skip the confirmation prompt")

	if err := flags.Parse(args); err != nil {
		// -h already printed usage; that is not a failure.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("parsing flags: %w", err)
	}

	if !*force {
		databaseConfig := configs.NewDatabaseConfig()
		prompt := fmt.Sprintf(
			"This rolls back and re-runs every migration in %s on %s:%s, destroying the data in those tables. Continue?",
			databaseConfig.Database, databaseConfig.Host, databaseConfig.Port,
		)

		confirmed, err := console.Confirm(os.Stdin, os.Stderr, prompt)
		if err != nil {
			return err
		}

		if !confirmed {
			log.Println("Aborted, nothing was changed")

			return nil
		}
	}

	db, err := core.NewPostgresService()
	if err != nil {
		return fmt.Errorf("connecting to the database: %w", err)
	}
	defer db.Close()

	log.Println("Rolling back and re-running all migrations...")

	if err := database.NewMigrator(db, database.NewKernel().Migrations).Refresh(); err != nil {
		return err
	}

	log.Println("Database refreshed")

	return nil
}

func (command *MigrateRefreshCommand) Description() string {
	return "Rolls back and re-runs every migration (--force skips confirmation)"
}
