package core

import (
	"database/sql"
	"log"
)

type SqliteService struct {
	db *sql.DB
}

// NewSqliteService initializes a new SqliteService with a database connection.
func NewSqliteService() (*SqliteService, error) {
	db, err := sql.Open("sqlite3", "./database/database.sqlite")
	if err != nil {
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
		return nil, err
	}
	defer stmt.Close()

	return stmt.Exec(args...)
}

// Read runs a SELECT query.
func (s *SqliteService) Read(query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	return stmt.Query(args...)
}

// Update runs an UPDATE query.
func (s *SqliteService) Update(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	return stmt.Exec(args...)
}

// Delete runs a DELETE query.
func (s *SqliteService) Delete(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	return stmt.Exec(args...)
}
