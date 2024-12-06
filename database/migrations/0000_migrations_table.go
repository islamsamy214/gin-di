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

	tx, _ := db.Begin()

	// Create the table
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			batch INTEGER DEFAULT 1
		);
	`)

	if err != nil {
		log.Printf("Failed to create migrations table: %v", err)
		tx.Rollback()
		return
	}

	// Create a record in the migrations table
	_, err = tx.Exec(`INSERT INTO migrations (name) VALUES ('migrations');`)
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

	log.Println("Migrations table created")
}

func (*Migrate) Down() {
	// Delete the table records
	log.Println("Dropping migrations table")

	// Initialize the service
	db, _ := core.NewSqliteService()

	result, _ := db.Delete(`DELETE FROM migrations;`)

	if result == nil {
		log.Println("Failed to drop migrations table")
	} else {
		log.Println("Migrations table dropped")
	}
}
