package seeders

import (
	"fmt"
	"web-app/app/models"
	"web-app/database/factories"
)

// seedEventCount is how many events the seeded account gets.
const seedEventCount = 3

type EventSeeder struct{}

func (e *EventSeeder) Run() error {
	// Events carry a foreign key, so the owner is looked up rather than
	// assuming the seeded user landed on id 1.
	owner := models.NewUserModel()
	owner.Username = seedUsername

	if err := owner.FindByUsername(); err != nil {
		return fmt.Errorf("finding user %s to own the seeded events: %w", seedUsername, err)
	}

	if _, err := factories.EventFactory().Count(seedEventCount).State(factories.ForUser(owner)).Create(); err != nil {
		return fmt.Errorf("creating events for %s: %w", seedUsername, err)
	}

	return nil
}
