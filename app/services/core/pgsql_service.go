package core

import (
	"database/sql"
	"log"
	"web-app/configs"

	_ "github.com/lib/pq"
)

type PostgresService struct {
	db *sql.DB
}

// NewPostgresService initializes a new PostgresService with a database connection.
func NewPostgresService() (*PostgresService, error) {
	databaseConfig := configs.NewDatabaseConfig()

	db, err := sql.Open(
		"postgres",
		"host="+databaseConfig.Host+" "+
			"port="+databaseConfig.Port+" "+
			"user="+databaseConfig.Username+" "+
			"password="+databaseConfig.Password+" "+
			"dbname="+databaseConfig.Database+" "+
			"sslmode=prefer")
	if err != nil {
		log.Println("Error opening database connection:", err)
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	log.Println("PostgreSQL database connection established")
	return &PostgresService{db: db}, nil
}

// Close closes the database connection.
func (s *PostgresService) Close() error {
	return s.db.Close()
}

// Create runs an INSERT query with a RETURNING clause.
func (s *PostgresService) Create(query string, args ...interface{}) (*sql.Row, error) {
	row := s.db.QueryRow(query, args...)
	return row, nil
}

// Read runs a SELECT query.
func (s *PostgresService) Read(query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		log.Println("Error preparing statement:", err)
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		log.Println("Error executing statement:", err)
		return nil, err
	}

	return rows, nil
}

// Update runs an UPDATE query.
func (s *PostgresService) Update(query string, args ...interface{}) (*sql.Row, error) {
	row := s.db.QueryRow(query, args...)
	return row, nil
}

// Delete runs a DELETE query.
func (s *PostgresService) Delete(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := s.db.Prepare(query)
	if err != nil {
		log.Println("Error preparing statement:", err)
		return nil, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(args...)
	if err != nil {
		log.Println("Error executing statement:", err)
		return nil, err
	}

	return res, nil
}

// Begin starts a transaction.
func (s *PostgresService) Begin() (*sql.Tx, error) {
	tx, err := s.db.Begin()
	if err != nil {
		log.Println("Error beginning transaction:", err)
		return nil, err
	}

	return tx, nil
}
