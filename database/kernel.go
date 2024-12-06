package database

import "web-app/database/migrations"

type kernel struct {
	Migrations map[string]migrations.Migration
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
	}

	return dbKernel
}
