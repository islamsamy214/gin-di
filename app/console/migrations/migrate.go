package migrations

import (
	"fmt"
	"log"
	"web-app/app/services/core"
	"web-app/database"
)

type MigrateCommand struct{}

func NewMigrateCommand() *MigrateCommand {
	return &MigrateCommand{}
}

/*
 * Migrate applies every migration that has not run yet.
 *
 * The command stays thin: opening the connection and reporting failure is its
 * job, while ordering and bookkeeping belong to the Migrator.
 */
func (command *MigrateCommand) Handle(args []string) error {
	db, err := core.NewPostgresService()
	if err != nil {
		return fmt.Errorf("connecting to the database: %w", err)
	}
	defer db.Close()

	log.Println("Migrating the database...")

	if err := database.NewMigrator(db, database.NewKernel().Migrations).Run(); err != nil {
		return err
	}

	log.Println("Database migrated")

	return nil
}

func (command *MigrateCommand) Description() string {
	return "Migrates the database"
}
