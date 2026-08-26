package resources

import "web-app/app/models"

/*
 * EventResource is the public projection of an event.
 *
 * The json names match models.Event's field for field. That is deliberate: it
 * keeps existing clients and the feature suite decoding unchanged while the
 * envelope around them moves, and it means adopting the resource is not itself a
 * breaking change to the object.
 */
type EventResource struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Date      string `json:"date"`
	CreatedAt string `json:"created_at"`
	UserID    int64  `json:"user_id"`
}

// NewEventResource projects a single event.
func NewEventResource(event *models.Event) EventResource {
	return EventResource{
		ID:        event.ID,
		Name:      event.Name,
		Date:      event.Date,
		CreatedAt: event.CreatedAt,
		UserID:    event.UserID,
	}
}

// SingleEvent is the payload of a response carrying one event.
type SingleEvent struct {
	Event EventResource `json:"event"`
}

// NewSingleEvent wraps one event as a response payload.
func NewSingleEvent(event *models.Event) SingleEvent {
	return SingleEvent{Event: NewEventResource(event)}
}

/*
 * NewEventCollection projects a page of events.
 *
 * The slice is allocated rather than declared, so an empty page marshals as []
 * and not null. A nil slice would make every client special-case "no results",
 * and some would crash on it.
 */
func NewEventCollection(events []models.Event) []EventResource {
	collection := make([]EventResource, 0, len(events))

	for index := range events {
		collection = append(collection, NewEventResource(&events[index]))
	}

	return collection
}
