package database

import (
	"web-app/app/interfaces"
	"web-app/database/migrations"
	"web-app/database/seeders"
)

/*
 * RegisteredMigration pairs a migration with the name stored in the migrations
 * table.
 */
type RegisteredMigration struct {
	Name      string
	Migration interfaces.Migration
}

/*
 * RegisteredSeeder pairs a seeder with the name used to select it.
 */
type RegisteredSeeder struct {
	Name   string
	Seeder interfaces.Seeder
}

type kernel struct {
	// A slice, not a map: migrations run in this order, so a table may
	// reference one declared above it. Map iteration is randomised in Go.
	Migrations []RegisteredMigration

	// Also a slice: a row referencing another must be seeded after it, and
	// map iteration is randomised in Go.
	Seeders []RegisteredSeeder
}

/*
 * NewKernel creates a new instance of the kernel
 */
func NewKernel() *kernel {
	dbKernel := &kernel{
		Migrations: []RegisteredMigration{
			// Add all the migrations here, in the order they must run.
			// {Name: "table_name", Migration: &migrations.MigrationStruct{}},
			// The migrations table itself is created by the Migrator.
			{Name: "users", Migration: &migrations.UserTable{}},
			{Name: "events", Migration: &migrations.EventTable{}},
		},

		Seeders: []RegisteredSeeder{
			// Add all the seeders here, in the order they must run.
			// {Name: "table_name", Seeder: &seeders.SeederStruct{}},
			{Name: "users", Seeder: &seeders.UserSeeder{}},
			{Name: "events", Seeder: &seeders.EventSeeder{}},
		},
	}

	return dbKernel
}
