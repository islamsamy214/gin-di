package console

import (
	"log"
	"slices"
	"web-app/app/services/core"
	"web-app/database"
	"web-app/database/migrations"
)

type MigrateCommand struct{}

func NewMigrateCommand() *MigrateCommand {
	return &MigrateCommand{}
}

/*
 * Migrate runs all the migrations
 */
func (command *MigrateCommand) Handle(args []string) error {
	var db, _ = core.NewPostgresService()

	log.Println("Migrating the database...")
	migrationsKernel := database.NewKernel()

	// Check the migrated tables
	migratedTables, err := db.Read(`SELECT name FROM migrations`)
	if err != nil {
		if err.Error() == `pq: relation "migrations" does not exist` {
			(&migrations.Migrate{}).Up()
			migratedTables, _ = db.Read(`SELECT name FROM migrations`)
		} else {
			log.Printf("Unexpected error: %v", err)
			return err
		}
	}

	var unmigratedTables []string
	for migratedTables.Next() {
		var tableName string
		migratedTables.Scan(&tableName)
		unmigratedTables = append(unmigratedTables, tableName)
	}

	for table, migration := range migrationsKernel.Migrations {
		if !slices.Contains(unmigratedTables, table) {
			migration.Up()
		}
	}

	log.Println("Database migrated")
	return nil // Ensure an explicit `nil` return on success
}

func (command *MigrateCommand) Description() string {
	return "Migrates the database"
}
