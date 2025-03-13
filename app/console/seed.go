package console

import (
	"log"
	"web-app/database"
)

type SeedCommand struct{}

func NewSeedCommand() *SeedCommand {
	return &SeedCommand{}
}

func (command *SeedCommand) Handle(args []string) error {
	log.Println("Starting the seed command")

	// Intialize the kernel
	seederKernel := database.NewKernel()

	for table, seeder := range seederKernel.Seeders {
		log.Printf("Seeding table %s", table)
		seeder.Run()
	}

	log.Println("Seed command completed")
	return nil
}

func (command *SeedCommand) Description() string {
	return "Seeds the database"
}
