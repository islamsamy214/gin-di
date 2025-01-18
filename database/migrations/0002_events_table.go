package migrations

import (
	"log"

	"web-app/app/services/core"
)

type EventTable struct{}

func (*EventTable) Up() {
	log.Println("Creating events table")

	// Initialize the service
	db, err := core.NewPostgresService()
	if err != nil {
		log.Printf("Failed to initialize database service: %v", err)
		return
	}
	defer db.Close()

	// Start the transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	// Create the table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			date DATE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			user_id INTEGER NOT NULL,
			CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Printf("Failed to create events table: %v", err)
		tx.Rollback()
		return
	}

	// Insert into migrations table
	_, err = tx.Exec(`INSERT INTO migrations (name) VALUES ($1);`, "events")
	if err != nil {
		log.Printf("Failed to insert into migrations table: %v", err)
		tx.Rollback()
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
	log.Println("Dropping events table")

	// Initialize the service
	db, err := core.NewPostgresService()
	if err != nil {
		log.Printf("Failed to initialize database service: %v", err)
		return
	}
	defer db.Close()

	// Start the transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	// Drop the table
	_, err = tx.Exec(`DROP TABLE IF EXISTS events;`)
	if err != nil {
		log.Printf("Failed to drop events table: %v", err)
		tx.Rollback()
		return
	}

	// Delete from migrations table
	_, err = tx.Exec(`DELETE FROM migrations WHERE name = $1;`, "events")
	if err != nil {
		log.Printf("Failed to delete from migrations table: %v", err)
		tx.Rollback()
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return
	}

	log.Println("Events table dropped")
}
