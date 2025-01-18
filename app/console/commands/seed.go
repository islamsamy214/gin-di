package commands

import (
	"log"
	"web-app/database"
)

func Seed() {
	log.Println("Starting the seed command")

	// Intialize the kernel
	seederKernel := database.NewKernel()

	for table, seeder := range seederKernel.Seeders {
		log.Printf("Seeding table %s", table)
		seeder.Run()
	}

	log.Println("Seed command completed")
}
