package migrations

import (
	"log"
	"os"
	"web-app/app/services/core"
)

type UserTable struct{}

func (*UserTable) Up() {
	log.Println("Creating users table")
	db, _ := core.NewSqliteService()

	// get the file "Migration" name
	migrationName := os.Args[1]

	// Begin transaction
	tx, _ := db.Begin()

	// Create the table
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT,
			password TEXT,
			created_at TEXT
		);
	`)
	if err != nil {
		log.Printf("Failed to create users table: %v", err)
		tx.Rollback()
		return
	}

	// Insert into migrations table
	_, err = tx.Exec(`INSERT INTO migrations (name) VALUES (?);`, migrationName)
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
	// Drop the table
	log.Println("Dropping users table")

	db, _ := core.NewSqliteService()

	// get the file "Migration" name
	migrationName := os.Args[1]

	// Begin transaction
	tx, _ := db.Begin()

	// Drop the table
	_, err := tx.Exec(`DROP TABLE IF EXISTS users;`)
	if err != nil {
		log.Printf("Failed to drop users table: %v", err)
		tx.Rollback()
		return
	}

	// Delete from migrations table
	_, err = tx.Exec(`DELETE FROM migrations WHERE name = ?;`, migrationName)
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
}
