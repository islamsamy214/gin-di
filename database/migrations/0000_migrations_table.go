package migrations

import (
	"log"

	"web-app/app/services/core"
)

type Migrate struct{}

func (*Migrate) Up() {
	log.Println("Creating migrations table")
	// Initialize the service
	sqliteService, err := core.NewSqliteService()
	if err != nil {
		log.Printf("Failed to initialize SqliteService: %v", err)
	}
	defer sqliteService.Close()

	_, err = sqliteService.Create(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			batch INTEGER
		);
	`)

	if err != nil {
		log.Printf("Failed to create migrations table: %v", err)
	}
	log.Println("Migrations table created")
}

func (*Migrate) Down() {
	log.Println("Dropping migrations table")
	// Initialize the service
	sqliteService, err := core.NewSqliteService()
	if err != nil {
		log.Printf("Failed to initialize SqliteService: %v", err)
	}
	defer sqliteService.Close()

	_, err = sqliteService.Delete(`
		DROP TABLE IF EXISTS migrations;
	`)

	if err != nil {
		log.Printf("Failed to drop migrations table: %v", err)
	}
}
