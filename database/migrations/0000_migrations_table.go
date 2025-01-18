package migrations

import (
	"log"

	"web-app/app/services/core"
)

type Migrate struct{}

func (*Migrate) Up() {
	log.Println("Creating migrations table")

	// Initialize the service
	db, err := core.NewPostgresService()
	if err != nil {
		log.Printf("Failed to initialize database service: %v", err)
		return
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	// Create the table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			batch INTEGER DEFAULT 1
		);
	`)
	if err != nil {
		log.Printf("Failed to create migrations table: %v", err)
		tx.Rollback()
		return
	}

	// Insert a record into the migrations table
	_, err = tx.Exec(`INSERT INTO migrations (name) VALUES ($1);`, "migrations")
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

	log.Println("Migrations table created")
}

func (*Migrate) Down() {
	log.Println("Dropping migrations table")

	// Initialize the service
	db, err := core.NewPostgresService()
	if err != nil {
		log.Printf("Failed to initialize database service: %v", err)
		return
	}
	defer db.Close()

	// Begin a transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	// Drop the table
	_, err = tx.Exec(`DROP TABLE IF EXISTS migrations;`)
	if err != nil {
		log.Printf("Failed to drop migrations table: %v", err)
		tx.Rollback()
		return
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return
	}

	log.Println("Migrations table dropped")
}
