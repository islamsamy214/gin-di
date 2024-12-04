package migrations

import (
	"log"

	"web-app/app/services/core"
)

type EventTable struct{}

func (*EventTable) Up() {
	// Create the events table
	log.Println("Creating events table")
	// Initialize the service
	sqliteService, err := core.NewSqliteService()
	if err != nil {
		log.Printf("Failed to initialize SqliteService: %v", err)
	}

	defer sqliteService.Close()

	_, err = sqliteService.Create(`
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			date TEXT,
			created_at TEXT,
			user_id INTEGER,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);
	`)
	if err != nil {
		log.Printf("Failed to create events table: %v", err)
	}
	log.Println("Events table created")
}

func (*EventTable) Down() {
	// Drop the table
	log.Println("Dropping events table")
	// Initialize the service
	sqliteService, err := core.NewSqliteService()
	if err != nil {
		log.Printf("Failed to initialize SqliteService: %v", err)
	}
	defer sqliteService.Close()

	_, err = sqliteService.Delete(`
		DROP TABLE IF EXISTS events;
	`)
	if err != nil {
		log.Printf("Failed to drop events table: %v", err)
	}
	log.Println("Events table dropped")
}
