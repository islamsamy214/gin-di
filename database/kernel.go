package database

import "web-app/database/migrations"

type kernel struct {
	Migrations []migrations.Migration
}

/*
 * NewKernel creates a new instance of the kernel
 */
func NewKernel() *kernel {
	dbKernel := &kernel{
		Migrations: []migrations.Migration{
			&migrations.Migrate{},
			&migrations.UserTable{},
			&migrations.EventTable{},
		},
	}

	return dbKernel
}
