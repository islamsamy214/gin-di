package commands

import (
	"log"
	"web-app/database"
)

func Migrate() {
	log.Println("Migrating the database...")
	// Initialize the kernel
	dbKernel := database.NewKernel()

	// Loop through all the migrations
	for _, migration := range dbKernel.Migrations {
		migration.Up()
	}

	log.Println("Database migrated")
}
