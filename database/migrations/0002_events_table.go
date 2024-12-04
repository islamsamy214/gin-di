package migrations

import (
	"log"
	"os"

	"web-app/app/services/core"
)

type EventTable struct{}

func (*EventTable) Up() {
	// Create the events table
	log.Println("Creating events table")

	// Initialize the service
	db, _ := core.NewSqliteService()

	// Start the transaction
	tx, _ := db.Begin()

	// Create the table
	_, err := tx.Exec(`
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
		tx.Rollback()
		return
	}

	// Get the migration name
	migrationName := os.Args[1]

	// Insert into migrations table
	_, err = tx.Exec(`INSERT INTO migrations (name) VALUES (?);`, migrationName)
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to insert into migrations table: %v", err)
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return
	}

	log.Println("Events table created")
}

func (*EventTable) Down() {
	// Drop the table
	log.Println("Dropping events table")

	// Initialize the service
	db, _ := core.NewSqliteService()

	// Start the transaction
	tx, _ := db.Begin()

	// Drop the table
	_, err := tx.Exec(`DROP TABLE IF EXISTS events;`)
	if err != nil {
		log.Printf("Failed to drop events table: %v", err)
		tx.Rollback()
		return
	}

	// Get the migration name
	migrationName := os.Args[1]

	// Delete from migrations table
	_, err = tx.Exec(`DELETE FROM migrations WHERE name = ?;`, migrationName)
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to delete from migrations table: %v", err)
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return
	}

	log.Println("Events table dropped")
}
