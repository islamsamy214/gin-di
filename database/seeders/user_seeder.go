package seeders

import (
	"fmt"
	"log"
	"web-app/app/models"
	"web-app/database/factories"
)

// seedUsername is the account a developer logs in as locally. Its password is
// factories.FactoryPassword.
const seedUsername = "islacks"

type UserSeeder struct{}

func (u *UserSeeder) Run() error {
	existing := models.NewUserModel()
	existing.Username = seedUsername

	// A lookup failure here means "not found", the normal path on a fresh
	// database, so it is deliberately not treated as an error.
	_ = existing.FindByUsername()

	if existing.ID != 0 {
		log.Printf("user %s already exists, skipping", seedUsername)

		return nil
	}

	// Username is pinned rather than faked: the whole point of this account is
	// that it can be logged into by name.
	if _, err := factories.UserFactory().State(factories.WithUsername(seedUsername)).CreateOne(); err != nil {
		return fmt.Errorf("creating user %s: %w", seedUsername, err)
	}

	return nil
}
