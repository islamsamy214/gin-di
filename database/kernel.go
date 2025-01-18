package database

import (
	"web-app/database/migrations"
	"web-app/database/seeders"
)

type kernel struct {
	Migrations map[string]migrations.Migration
	Seeders    map[string]seeders.Seeder
}

/*
 * NewKernel creates a new instance of the kernel
 */
func NewKernel() *kernel {
	dbKernel := &kernel{
		Migrations: map[string]migrations.Migration{
			// Add all the migrations here
			// "table_name": &migrations.MigrationStruct{},
			"migrations": &migrations.Migrate{},
			"users":      &migrations.UserTable{},
			"events":     &migrations.EventTable{},
		},

		Seeders: map[string]seeders.Seeder{
			// Add all the seeders here
			// "table_name": &seeders.SeederStruct{},
			"users":  &seeders.UserSeeder{},
			"events": &seeders.EventSeeder{},
		},
	}

	return dbKernel
}
