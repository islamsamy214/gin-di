package models

import (
	"database/sql"
	"errors"
	"log"
	"web-app/app/services/core"
)

type Event struct {
	ID        int64  `json:"id"`
	Name      string `form:"name" json:"name" xml:"name" binding:"required"`
	Date      string `form:"date" json:"date" xml:"date" binding:"required"`
	CreatedAt string `json:"created_at"`
	UserId    int64  `json:"user_id"`
	db        *core.PostgresService
}

func NewEventModel() *Event {
	db, _ := core.NewPostgresService()
	return &Event{
		db: db,
	}
}

// Create inserts a new event into the database
func (event *Event) Create() error {
	query := `
        INSERT INTO events (name, date, user_id)
        VALUES ($1, $2, $3)
        RETURNING id, created_at`

	result, err := event.db.Create(query, event.Name, event.Date, event.UserId)
	if err != nil {
		log.Printf("error creating event: %v", err)
		return err
	}

	// Get the ID and CreatedAt from the result
	err = result.Scan(&event.ID, &event.CreatedAt)
	if err != nil {
		log.Printf("error scanning event: %v", err)
	}

	return nil
}

// Find retrieves an event by its ID
func (event *Event) Find() error {
	if event.ID == 0 {
		return errors.New("id is required")
	}

	query := `
        SELECT id, name, date, created_at, user_id 
        FROM events 
        WHERE id = $1`

	rows, err := event.db.Read(query, event.ID)
	if err != nil {
		log.Printf("error finding event: %v", err)
		return err
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.Scan(&event.ID, &event.Name, &event.Date, &event.CreatedAt, &event.UserId)
		if err != nil {
			log.Printf("error scanning event: %v", err)
			return err
		}
		return nil
	}

	return sql.ErrNoRows
}

// Update modifies an existing event in the database
func (event *Event) Update() error {
	if event.ID == 0 {
		return errors.New("id is required")
	}

	query := `
        UPDATE events 
        SET name = $1, date = $2, user_id = $3
        WHERE id = $4`

	_, err := event.db.Update(query, event.Name, event.Date, event.UserId, event.ID)
	if err != nil {
		log.Printf("error updating event: %v", err)
		return err
	}

	return nil
}

// Delete removes an event from the database
func (event *Event) Delete() error {
	if event.ID == 0 {
		return errors.New("id is required")
	}

	query := `
        DELETE FROM events 
        WHERE id = $1`

	_, err := event.db.Delete(query, event.ID)
	if err != nil {
		log.Printf("error deleting event: %v", err)
		return err
	}

	return nil
}

// Paginate retrieves a paginated list of events
func (event *Event) Paginate(limit, page int) ([]Event, error) {
	// Set default values
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit
	query := `
        SELECT id, name, date, created_at, user_id
        FROM events
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2`

	rows, err := event.db.Read(query, limit, offset)
	if err != nil {
		log.Printf("error paginating events: %v", err)
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0, limit)
	for rows.Next() {
		var event Event
		err := rows.Scan(&event.ID, &event.Name, &event.Date, &event.CreatedAt, &event.UserId)
		if err != nil {
			log.Printf("error scanning event: %v", err)
			return nil, err
		}
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		log.Printf("error iterating over rows: %v", err)
		return nil, err
	}

	return events, nil
}
