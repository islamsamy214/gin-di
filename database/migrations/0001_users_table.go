package migrations

import (
	"log"
	"web-app/app/services/core"
)

type UserTable struct{}

func (*UserTable) Up() {
	log.Println("Creating users table")

	// Initialize the service
	db, err := core.NewPostgresService()
	if err != nil {
		log.Printf("Failed to initialize database service: %v", err)
		return
	}
	defer db.Close()

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	// Create the table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Printf("Failed to create users table: %v", err)
		tx.Rollback()
		return
	}

	// Insert into migrations table
	_, err = tx.Exec(`INSERT INTO migrations (name) VALUES ($1);`, "users")
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to insert into migrations table: %v", err)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return
	}

	log.Println("Users table created")
}

func (*UserTable) Down() {
	log.Println("Dropping users table")

	// Initialize the service
	db, err := core.NewPostgresService()
	if err != nil {
		log.Printf("Failed to initialize database service: %v", err)
		return
	}
	defer db.Close()

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	// Drop the table
	_, err = tx.Exec(`DROP TABLE IF EXISTS users;`)
	if err != nil {
		log.Printf("Failed to drop users table: %v", err)
		tx.Rollback()
		return
	}

	// Delete from migrations table
	_, err = tx.Exec(`DELETE FROM migrations WHERE name = $1;`, "users")
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to delete from migrations table: %v", err)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return
	}

	log.Println("Users table dropped")
}
