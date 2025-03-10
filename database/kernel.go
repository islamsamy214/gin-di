package database

import (
	"web-app/app/interfaces"
	"web-app/database/migrations"
	"web-app/database/seeders"
)

type kernel struct {
	Migrations map[string]interfaces.Migration
	Seeders    map[string]interfaces.Seeder
}

/*
 * NewKernel creates a new instance of the kernel
 */
func NewKernel() *kernel {
	dbKernel := &kernel{
		Migrations: map[string]interfaces.Migration{
			// Add all the migrations here
			// "table_name": &migrations.MigrationStruct{},
			"migrations": &migrations.Migrate{},
			"users":      &migrations.UserTable{},
			"events":     &migrations.EventTable{},
		},

		Seeders: map[string]interfaces.Seeder{
			// Add all the seeders here
			// "table_name": &seeders.SeederStruct{},
			"users":  &seeders.UserSeeder{},
			"events": &seeders.EventSeeder{},
		},
	}

	return dbKernel
}
