package commands

import (
	"log"
	"web-app/app/helpers"
	"web-app/app/services/core"
	"web-app/database"
	"web-app/database/migrations"
)

/*
 * Migrate runs all the migrations
 */
func Migrate() {
	var db, _ = core.NewPostgresService()

	log.Println("Migrating the database...")
	// Initialize the kernel
	migrationsKernel := database.NewKernel()

	// Check the migrated tables
	migratedTables, err := db.Read(`SELECT name FROM migrations`)

	// Check if migrations table does not exist create it
	if err != nil {
		if err.Error() == `pq: relation "migrations" does not exist` {
			// Create the migrations table
			(&migrations.Migrate{}).Up()
			migratedTables, _ = db.Read(`SELECT name FROM migrations`)
		} else {
			log.Printf("Unexpected error: %v", err)
		}
	}

	// Get the un-migrated tables
	var unmigratedTables []string
	for migratedTables.Next() {
		var tableName string
		migratedTables.Scan(&tableName)
		unmigratedTables = append(unmigratedTables, tableName)
	}

	// Loop through all the migrations
	for table, migration := range migrationsKernel.Migrations {
		// Check if the migration has been run
		if !helpers.Contains(unmigratedTables, table) {
			migration.Up()
		}

	}

	log.Println("Database migrated")
}

/*
 * Rollback rolls back all the migrations
 */
func Rollback(args []string) {
	log.Println("Rolling back the database...")
	// Initialize the kernel
	dbKernel := database.NewKernel()

	// if args[4] is set, then rollback only that migration
	if len(args) > 3 {
		for table, migration := range dbKernel.Migrations {
			if table == args[3] {
				migration.Down()
			}
		}
		return
	}

	// Loop through all the migrations in reverse order
	for _, migration := range dbKernel.Migrations {
		migration.Down()
	}

	log.Println("Database rolled back")
}
