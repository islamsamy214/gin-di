package migrations

import "database/sql"

type EventTable struct{}

func (*EventTable) Up(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			date DATE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			user_id INTEGER NOT NULL,
			CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)

	return err
}

func (*EventTable) Down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS events;`)

	return err
}
