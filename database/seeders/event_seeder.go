package seeders

import (
	"log"
	"web-app/app/models"
)

type EventSeeder struct{}

func (e *EventSeeder) Run() {
	eventModel := models.NewEventModel()

	randomEvents := getRandomEvents()

	for _, event := range randomEvents {
		eventModel.Name = event.Name
		eventModel.Date = event.Date
		eventModel.UserId = event.UserId
		err := eventModel.Create()
		if err != nil {
			log.Fatalf("error creating event: %v", err)
		}
	}
}

func getRandomEvents() []*models.Event {
	events := []*models.Event{
		{
			Name:   "Birthday Party",
			Date:   "2025-02-15",
			UserId: 1,
		},
		{
			Name:   "Test Event",
			Date:   "2025-02-16",
			UserId: 1,
		},
		{
			Name:   "Another Event",
			Date:   "2025-02-17",
			UserId: 1,
		},
	}

	return events
}
