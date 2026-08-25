package interfaces

import "database/sql"

type Migration interface {
	// Up applies the schema change inside the migrator's transaction
	Up(tx *sql.Tx) error

	// Down reverses the schema change inside the migrator's transaction
	Down(tx *sql.Tx) error
}
