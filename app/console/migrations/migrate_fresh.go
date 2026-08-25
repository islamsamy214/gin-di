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

type MigrateFreshCommand struct{}

func NewMigrateFreshCommand() *MigrateFreshCommand {
	return &MigrateFreshCommand{}
}

/*
 * Handle drops every table in the database and migrates from scratch.
 *
 * This destroys all data, so it names the target and waits for confirmation
 * unless --force is passed.
 */
func (command *MigrateFreshCommand) Handle(args []string) error {
	flags := flag.NewFlagSet("migrate:fresh", flag.ContinueOnError)
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
			"This DROPS EVERY TABLE in %s on %s:%s and destroys all data. Continue?",
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

	log.Println("Dropping all tables and migrating from scratch...")

	if err := database.NewMigrator(db, database.NewKernel().Migrations).Fresh(); err != nil {
		return err
	}

	log.Println("Database built from scratch")

	return nil
}

func (command *MigrateFreshCommand) Description() string {
	return "Drops all tables and re-runs every migration (--force skips confirmation)"
}
