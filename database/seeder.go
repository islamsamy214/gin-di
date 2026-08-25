package database

import (
	"fmt"
	"log"
	"strings"
)

/*
 * DatabaseSeeder runs the registered seeders.
 *
 * Mirrors Laravel's DatabaseSeeder: one entry point that calls each seeder in
 * a declared order, so no caller has to know the dependencies between them.
 */
type DatabaseSeeder struct {
	seeders []RegisteredSeeder
}

func NewDatabaseSeeder(seeders []RegisteredSeeder) *DatabaseSeeder {
	return &DatabaseSeeder{
		seeders: seeders,
	}
}

/*
 * Run seeds every registered seeder, in registration order.
 *
 * Stops at the first failure rather than carrying on into seeders whose rows
 * depend on the one that just failed.
 *
 * @return error The first seeding failure, or nil.
 */
func (databaseSeeder *DatabaseSeeder) Run() error {
	for _, registered := range databaseSeeder.seeders {
		if err := databaseSeeder.call(registered); err != nil {
			return err
		}
	}

	return nil
}

/*
 * RunClass seeds a single registered seeder, as `db:seed --class` does.
 *
 * @return error If the name is unknown, or the seeder fails.
 */
func (databaseSeeder *DatabaseSeeder) RunClass(name string) error {
	for _, registered := range databaseSeeder.seeders {
		if registered.Name == name {
			return databaseSeeder.call(registered)
		}
	}

	return fmt.Errorf("seeder %q is not registered, known seeders: %s", name, strings.Join(databaseSeeder.Names(), ", "))
}

/*
 * Names lists the registered seeder names, for help text and error messages.
 *
 * @return []string The names, in registration order.
 */
func (databaseSeeder *DatabaseSeeder) Names() []string {
	names := make([]string, 0, len(databaseSeeder.seeders))

	for _, registered := range databaseSeeder.seeders {
		names = append(names, registered.Name)
	}

	return names
}

/*
 * call runs one seeder and reports it, the way Laravel's Seeder::call does.
 *
 * @return error The seeding failure, wrapped with the seeder name.
 */
func (databaseSeeder *DatabaseSeeder) call(registered RegisteredSeeder) error {
	log.Printf("Seeding: %s", registered.Name)

	if err := registered.Seeder.Run(); err != nil {
		return fmt.Errorf("seeding %s: %w", registered.Name, err)
	}

	log.Printf("Seeded:  %s", registered.Name)

	return nil
}
