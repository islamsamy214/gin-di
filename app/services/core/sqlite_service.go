package core

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type SqliteService struct {
	db *sql.DB
}

// NewSqliteService initializes a new SqliteService with a database connection.
func NewSqliteService() (*SqliteService, error) {
	db, err := sql.Open("sqlite3", "./database/database.sqlite")
	if err != nil {
		log.Println("Error opening database connection", err)
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	log.Println("Database connection established")
	return &SqliteService{db: db}, nil
}

// Close closes the database connection.
func (s *SqliteService) Close() error {
	return s.db.Close()
}

// Create runs an INSERT query.
func (s *SqliteService) Create(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		log.Println("Error preparing statement", err)
		return nil, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(args...)
	if err != nil {
		log.Println("Error executing statement", err)
		return nil, err
	}

	return res, nil
}

// Read runs a SELECT query.
func (s *SqliteService) Read(query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		log.Println("Error preparing statement", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		log.Println("Error executing statement", err)
		return nil, err
	}

	return rows, nil
}

// Update runs an UPDATE query.
func (s *SqliteService) Update(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		log.Println("Error preparing statement", err)
		return nil, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(args...)
	if err != nil {
		log.Println("Error executing statement", err)
		return nil, err
	}

	return res, nil
}

// Delete runs a DELETE query.
func (s *SqliteService) Delete(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		log.Println("Error preparing statement", err)
		return nil, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(args...)
	if err != nil {
		log.Println("Error executing statement", err)
		return nil, err
	}

	return res, nil
}

// Begin starts a transaction.
func (s *SqliteService) Begin() (*sql.Tx, error) {
	tx, err := s.db.Begin()
	if err != nil {
		log.Println("Error beginning transaction", err)
		return nil, err
	}

	return tx, nil
}
