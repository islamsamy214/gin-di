package seeders

import (
	"log"
	"web-app/app/models"
	"web-app/app/services"
)

type UserSeeder struct{}

func (u *UserSeeder) Run() {
	userModel := models.NewUserModel()

	userModel.Username = "islacks"
	hashedPassword, err := services.HashPassword("password")
	if err != nil {
		log.Fatalf("error hashing password: %v", err)
	}
	userModel.Password = hashedPassword

	// Check if the user already exists
	userModel.FindByUsername()
	if userModel.ID != 0 {
		log.Println("user already exists")
		return
	}

	err = userModel.Create()
	if err != nil {
		log.Fatalf("error creating user: %v", err)
	}
}
