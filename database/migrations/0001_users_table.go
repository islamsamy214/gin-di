package migrations

import (
	"log"
	"web-app/app/services/core"
)

type UserTable struct{}

func (*UserTable) Up() {
	log.Println("Creating users table")
	sqliteService, err := core.NewSqliteService()
	if err != nil {
		log.Printf("Failed to initialize SqliteService: %v", err)
	}
	defer sqliteService.Close()

	_, err = sqliteService.Create(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			password TEXT,
			created_at TEXT
		);
	`)

	if err != nil {
		log.Printf("Failed to create users table: %v", err)
	}
	log.Println("Users table created")
}

func (*UserTable) Down() {
	// Drop the table
	log.Println("Dropping users table")
	// Initialize the service
	sqliteService, err := core.NewSqliteService()
	if err != nil {
		log.Printf("Failed to initialize SqliteService: %v", err)
	}
	defer sqliteService.Close()

	_, err = sqliteService.Delete(`
		DROP TABLE IF EXISTS users;
	`)
	if err != nil {
		log.Printf("Failed to drop users table: %v", err)
	}
	log.Println("Users table dropped")
}
