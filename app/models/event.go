package models

import (
	"database/sql"
	"errors"
	"fmt"
	"web-app/app/services/core"
)

// defaultPageSize is used when a caller passes a non-positive limit.
const defaultPageSize = 10

/*
 * Event is a row of the events table.
 *
 * The form, xml and binding tags that used to sit on Name and Date have moved to
 * requests.StoreRequest. Binding directly onto this struct made UserID
 * mass-assignable from a request body, which is the root of the ownership bug the
 * controller papered over by hardcoding an id.
 */
type Event struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Date      string `json:"date"`
	CreatedAt string `json:"created_at"`
	UserID    int64  `json:"user_id"`
	db        *core.PostgresService
}

// NewEventModel binds a model to the shared connection. See NewUserModel for why
// this resolves rather than opens.
func NewEventModel() *Event {
	db, _ := core.Connection()

	return &Event{db: db}
}

// Create inserts the event and reads back its generated columns.
func (event *Event) Create() error {
	query := `
        INSERT INTO events (name, date, user_id)
        VALUES ($1, $2, $3)
        RETURNING id, created_at`

	result, err := event.db.Create(query, event.Name, event.Date, event.UserID)
	if err != nil {
		return fmt.Errorf("creating event: %w", err)
	}

	// QueryRow defers the insert error to Scan, so returning nil here hid
	// every constraint violation from the caller.
	if err := result.Scan(&event.ID, &event.CreatedAt); err != nil {
		return fmt.Errorf("scanning created event: %w", err)
	}

	return nil
}

// Find loads the event matching the set id.
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
		return fmt.Errorf("finding event: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&event.ID, &event.Name, &event.Date, &event.CreatedAt, &event.UserID); err != nil {
			return fmt.Errorf("scanning event: %w", err)
		}

		return nil
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating events: %w", err)
	}

	return sql.ErrNoRows
}

// Update writes the model's current values back to its row.
func (event *Event) Update() error {
	if event.ID == 0 {
		return errors.New("id is required")
	}

	query := `
        UPDATE events
        SET name = $1, date = $2, user_id = $3
        WHERE id = $4`

	if _, err := event.db.Update(query, event.Name, event.Date, event.UserID, event.ID); err != nil {
		return fmt.Errorf("updating event: %w", err)
	}

	return nil
}

// Delete removes the model's row.
func (event *Event) Delete() error {
	if event.ID == 0 {
		return errors.New("id is required")
	}

	query := `
        DELETE FROM events
        WHERE id = $1`

	if _, err := event.db.Delete(query, event.ID); err != nil {
		return fmt.Errorf("deleting event: %w", err)
	}

	return nil
}

/*
 * Paginate lists every event, newest first, regardless of owner.
 *
 * Unscoped, and must not be called from an HTTP handler: use PaginateForUser.
 * Filtering in the handler after an unscoped read is how a WHERE clause gets
 * forgotten, and forgetting this one exposed every user's events to every
 * authenticated caller. This exists for console tooling and tests, where there is
 * no "current user" to scope to.
 *
 * @param limit Rows per page; defaults to 10 when not positive.
 * @param page  One-based page number; defaults to 1 when not positive.
 */
func (event *Event) Paginate(limit, page int) ([]Event, error) {
	query := `
        SELECT id, name, date, created_at, user_id
        FROM events
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2`

	limit, offset := pageBounds(limit, page)

	rows, err := event.db.Read(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("paginating events: %w", err)
	}

	return scanEvents(rows, limit)
}

/*
 * PaginateForUser lists one owner's events, newest first.
 *
 * The owner-scoped counterpart to Paginate, and the only one an HTTP handler
 * should reach for.
 *
 * @param userID The owner whose events are returned.
 * @param limit  Rows per page; defaults to 10 when not positive.
 * @param page   One-based page number; defaults to 1 when not positive.
 */
func (event *Event) PaginateForUser(userID int64, limit, page int) ([]Event, error) {
	query := `
        SELECT id, name, date, created_at, user_id
        FROM events
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3`

	limit, offset := pageBounds(limit, page)

	rows, err := event.db.Read(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("paginating events for user %d: %w", userID, err)
	}

	return scanEvents(rows, limit)
}

/*
 * pageBounds normalises a limit and page into a limit and offset.
 *
 * @return int The row limit.
 * @return int The row offset.
 */
func pageBounds(limit, page int) (int, int) {
	if limit <= 0 {
		limit = defaultPageSize
	}

	if page <= 0 {
		page = 1
	}

	return limit, (page - 1) * limit
}

/*
 * scanEvents drains a result set into events and closes it.
 *
 * Shared by both paginators so the scan loop and its rows.Err() check exist once
 * — the alternative is two copies that drift, and the one that drifts is the one
 * that silently drops the error.
 *
 * @param rows     The result set, consumed and closed here.
 * @param capacity The expected row count, used to size the slice.
 */
func scanEvents(rows *sql.Rows, capacity int) ([]Event, error) {
	defer rows.Close()

	events := make([]Event, 0, capacity)

	for rows.Next() {
		var found Event

		if err := rows.Scan(&found.ID, &found.Name, &found.Date, &found.CreatedAt, &found.UserID); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}

		events = append(events, found)
	}

	// Without this a failure part way through iteration looks like a clean
	// finish, and a partial page reads as a complete one.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating events: %w", err)
	}

	return events, nil
}
