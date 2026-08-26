package factories

import (
	"time"
	"web-app/app/models"

	"github.com/brianvoe/gofakeit/v7"
)

// dateLayout matches the DATE column the events table declares.
const dateLayout = "2006-01-02"

/*
 * EventFactory builds events.
 *
 * UserID is deliberately left unset: events carry a foreign key to users, so
 * the caller states an owner with ForUser rather than the factory inventing
 * one. A bare Create therefore fails loudly instead of guessing an id.
 *
 * @return *Factory[*models.Event] A fluent factory.
 */
func EventFactory() *Factory[*models.Event] {
	return New(func(sequence int) (*models.Event, error) {
		event := models.NewEventModel()
		event.Name = gofakeit.Dinner()

		// Spread across distinct future days so ordering is observable.
		event.Date = time.Now().AddDate(0, 0, sequence).Format(dateLayout)

		return event, nil
	})
}

/*
 * ForUser ties built events to an existing user, like Laravel's ->for().
 *
 * @return func(*models.Event) A state for Factory.State.
 */
func ForUser(user *models.User) func(*models.Event) {
	return func(event *models.Event) {
		event.UserID = user.ID
	}
}
