package migrations

import (
	"log"

	"web-app/app/services/core"
)

type Migrate struct{}

func (*Migrate) Up() {
	log.Println("Creating migrations table")
	// Initialize the service
	db, _ := core.NewSqliteService()

	result, _ := db.Create(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			batch INTEGER DEFAULT 1
		);
	`)

	if result == nil {
		log.Println("Failed to create migrations table")
	} else {
		log.Println("Migrations table created")
	}
}

func (*Migrate) Down() {
	log.Println("Dropping migrations table")
	// Initialize the service
	db, _ := core.NewSqliteService()

	result, _ := db.Create("DROP TABLE IF EXISTS migrations;")

	if result == nil {
		log.Println("Failed to drop migrations table")
	} else {
		log.Println("Migrations table dropped")
	}
}
