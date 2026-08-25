package migrations

import "database/sql"

type UserTable struct{}

func (*UserTable) Up(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)

	return err
}

func (*UserTable) Down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS users;`)

	return err
}
