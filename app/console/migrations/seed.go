package migrations

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"web-app/app/console"
	"web-app/configs"
	"web-app/database"
)

// productionEnv is the APP_ENV value that makes seeding ask first.
const productionEnv = "production"

type SeedCommand struct{}

func NewSeedCommand() *SeedCommand {
	return &SeedCommand{}
}

/*
 * Handle seeds the database.
 *
 * Runs every registered seeder in order, or one of them with --class. Seeding
 * a production database writes fixture rows into live tables, so that case
 * asks for confirmation unless --force is given.
 */
func (command *SeedCommand) Handle(args []string) error {
	databaseSeeder := database.NewDatabaseSeeder(database.NewKernel().Seeders)

	flags := flag.NewFlagSet("db:seed", flag.ContinueOnError)
	class := flags.String("class", "", "seed only this seeder ("+strings.Join(databaseSeeder.Names(), ", ")+")")
	force := flags.Bool("force", false, "skip the confirmation when APP_ENV is "+productionEnv)

	if err := flags.Parse(args); err != nil {
		// -h already printed usage; that is not a failure.
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return fmt.Errorf("parsing flags: %w", err)
	}

	if configs.NewAppConfig().Env == productionEnv && !*force {
		prompt := fmt.Sprintf("APP_ENV is %s. Seeding writes fixture data into live tables. Continue?", productionEnv)

		confirmed, err := console.Confirm(os.Stdin, os.Stderr, prompt)
		if err != nil {
			return err
		}

		if !confirmed {
			log.Println("Aborted, nothing was seeded")

			return nil
		}
	}

	log.Println("Seeding the database...")

	if *class != "" {
		if err := databaseSeeder.RunClass(*class); err != nil {
			return err
		}

		log.Println("Database seeded")

		return nil
	}

	if err := databaseSeeder.Run(); err != nil {
		return err
	}

	log.Println("Database seeded")

	return nil
}

func (command *SeedCommand) Description() string {
	return "Seeds the database (--class runs one seeder, --force skips the production prompt)"
}
