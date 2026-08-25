package database

import (
	"fmt"
	"log"
	"web-app/app/services/core"

	"github.com/lib/pq"
)

/*
 * Migrator applies pending migrations and records them.
 *
 * Mirrors Laravel's Migrator: a migration file describes only its schema
 * change, while ordering, transactions and the migrations table itself are
 * owned here. That keeps the bookkeeping in one place instead of copied into
 * every migration.
 */
type Migrator struct {
	db         *core.PostgresService
	migrations []RegisteredMigration
}

func NewMigrator(db *core.PostgresService, migrations []RegisteredMigration) *Migrator {
	return &Migrator{
		db:         db,
		migrations: migrations,
	}
}

/*
 * Run applies every registered migration that has not been recorded yet.
 *
 * Migrations run in registration order, so a table may safely reference one
 * declared before it.
 *
 * @return error The first migration failure, or nil.
 */
func (migrator *Migrator) Run() error {
	if err := migrator.ensureRepository(); err != nil {
		return err
	}

	applied, err := migrator.appliedMigrations()
	if err != nil {
		return err
	}

	batch, err := migrator.nextBatch()
	if err != nil {
		return err
	}

	ran := 0

	for _, registered := range migrator.migrations {
		if _, alreadyApplied := applied[registered.Name]; alreadyApplied {
			continue
		}

		log.Printf("Migrating: %s", registered.Name)

		if err := migrator.apply(registered, batch); err != nil {
			return fmt.Errorf("migrating %s: %w", registered.Name, err)
		}

		log.Printf("Migrated:  %s", registered.Name)

		ran++
	}

	if ran == 0 {
		log.Println("Nothing to migrate")
	}

	return nil
}

/*
 * apply runs one migration and records it in the same transaction.
 *
 * Sharing the transaction means the schema change and its migrations-table row
 * cannot diverge: a failure leaves neither behind.
 *
 * @return error The failure that aborted the migration, or nil.
 */
func (migrator *Migrator) apply(registered RegisteredMigration, batch int) error {
	tx, err := migrator.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	if err := registered.Migration.Up(tx); err != nil {
		// The rollback outcome is secondary to the failure that triggered it.
		_ = tx.Rollback()

		return err
	}

	if _, err := tx.Exec(`INSERT INTO migrations (name, batch) VALUES ($1, $2);`, registered.Name, batch); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("recording migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

/*
 * ensureRepository creates the migrations table when it is absent.
 *
 * Doing this unconditionally replaces the previous approach of matching the
 * driver's error text to detect a fresh database, which never matched because
 * lib/pq appends the column position and SQLSTATE to the message.
 *
 * @return error If the table cannot be created.
 */
func (migrator *Migrator) ensureRepository() error {
	tx, err := migrator.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			batch INTEGER NOT NULL DEFAULT 1
		);
	`); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("creating migrations table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

/*
 * appliedMigrations reads the names already recorded as run.
 *
 * @return map[string]struct{} The set of applied migration names.
 * @return error               If the rows cannot be read.
 */
func (migrator *Migrator) appliedMigrations() (map[string]struct{}, error) {
	rows, err := migrator.db.Read(`SELECT name FROM migrations`)
	if err != nil {
		return nil, fmt.Errorf("reading applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]struct{})

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning applied migration: %w", err)
		}

		applied[name] = struct{}{}
	}

	// Without this a failure part way through iteration looks like a clean
	// finish, and every remaining migration would be re-applied.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating applied migrations: %w", err)
	}

	return applied, nil
}

/*
 * nextBatch returns the batch number this run should record.
 *
 * Grouping a run's migrations under one batch is what makes rolling back a
 * single deployment possible.
 *
 * @return int   The next batch number.
 * @return error If the current batch cannot be read.
 */
func (migrator *Migrator) nextBatch() (int, error) {
	rows, err := migrator.db.Read(`SELECT COALESCE(MAX(batch), 0) FROM migrations`)
	if err != nil {
		return 0, fmt.Errorf("reading the last batch: %w", err)
	}
	defer rows.Close()

	batch := 0

	if rows.Next() {
		if err := rows.Scan(&batch); err != nil {
			return 0, fmt.Errorf("scanning the last batch: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterating the last batch: %w", err)
	}

	return batch + 1, nil
}

/*
 * Fresh drops every table in the database and runs all migrations again.
 *
 * Drops tables outright rather than calling Down(), so a database left broken
 * by a half-applied migration can still be rebuilt.
 *
 * @return error The first failure, or nil.
 */
func (migrator *Migrator) Fresh() error {
	if err := migrator.dropAllTables(); err != nil {
		return err
	}

	return migrator.Run()
}

/*
 * Refresh rolls every applied migration back, then runs them again.
 *
 * Unlike Fresh this goes through each migration's Down(), so it also proves the
 * rollback path works.
 *
 * @return error The first failure, or nil.
 */
func (migrator *Migrator) Refresh() error {
	if err := migrator.Reset(); err != nil {
		return err
	}

	return migrator.Run()
}

/*
 * Reset rolls back every applied migration, newest first.
 *
 * Reverse registration order matters: a table must be dropped before the one
 * it references.
 *
 * @return error The first rollback failure, or nil.
 */
func (migrator *Migrator) Reset() error {
	if err := migrator.ensureRepository(); err != nil {
		return err
	}

	applied, err := migrator.appliedMigrations()
	if err != nil {
		return err
	}

	rolledBack := 0

	for i := len(migrator.migrations) - 1; i >= 0; i-- {
		registered := migrator.migrations[i]

		if _, isApplied := applied[registered.Name]; !isApplied {
			continue
		}

		log.Printf("Rolling back: %s", registered.Name)

		if err := migrator.revert(registered); err != nil {
			return fmt.Errorf("rolling back %s: %w", registered.Name, err)
		}

		log.Printf("Rolled back: %s", registered.Name)

		rolledBack++
	}

	if rolledBack == 0 {
		log.Println("Nothing to roll back")
	}

	return nil
}

/*
 * revert rolls one migration back and forgets it in the same transaction.
 *
 * @return error The failure that aborted the rollback, or nil.
 */
func (migrator *Migrator) revert(registered RegisteredMigration) error {
	tx, err := migrator.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	if err := registered.Migration.Down(tx); err != nil {
		// The rollback outcome is secondary to the failure that triggered it.
		_ = tx.Rollback()

		return err
	}

	if _, err := tx.Exec(`DELETE FROM migrations WHERE name = $1;`, registered.Name); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("forgetting migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

/*
 * dropAllTables removes every base table in the current schema.
 *
 * CASCADE means drop order does not matter, which is what lets Fresh recover a
 * database whose foreign keys are in an unknown state.
 *
 * @return error If any table cannot be dropped.
 */
func (migrator *Migrator) dropAllTables() error {
	tables, err := migrator.tableNames()
	if err != nil {
		return err
	}

	if len(tables) == 0 {
		log.Println("No tables to drop")

		return nil
	}

	tx, err := migrator.db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	for _, table := range tables {
		log.Printf("Dropping: %s", table)

		// Quoted: the identifier comes from the catalogue, not a literal.
		if _, err := tx.Exec(`DROP TABLE IF EXISTS ` + pq.QuoteIdentifier(table) + ` CASCADE;`); err != nil {
			_ = tx.Rollback()

			return fmt.Errorf("dropping %s: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

/*
 * tableNames lists the base tables in the schema the connection is using.
 *
 * @return []string The table names.
 * @return error    If the catalogue cannot be read.
 */
func (migrator *Migrator) tableNames() ([]string, error) {
	rows, err := migrator.db.Read(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = current_schema()
		  AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		return nil, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()

	var tables []string

	for rows.Next() {
		var table string

		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("scanning table name: %w", err)
		}

		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tables: %w", err)
	}

	return tables, nil
}
