package models

import (
	"database/sql"
	"errors"
	"fmt"
	"web-app/app/services/core"
)

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`

	/*
	 * json:"-" in both directions, deliberately.
	 *
	 * Outbound, this is the stored argon2id hash and must never be marshalled to
	 * a client — it was, on every successful login, because the model was
	 * returned as the response body. Inbound, a password must never arrive on a
	 * model: credentials bind to requests.LoginRequest and are handled by
	 * UserService.
	 *
	 * The HTTP validation tags that used to sit on these fields have gone too. A
	 * persistence struct is the wrong place for request rules, and carrying them
	 * is what made one type serve as row, bind target and response at once.
	 */
	Password string `json:"-"`

	CreatedAt string `json:"created_at"`
	db        *core.PostgresService
}

/*
 * NewUserModel binds a model to the shared connection.
 *
 * core.Connection rather than core.NewPostgresService: the latter opened a fresh
 * pool per call and the error was discarded, so every request leaked ten
 * connections and a database outage produced a nil handle that panicked on first
 * use. The error is still not returned here — the shared pool is opened and
 * validated once at boot, so by the time any model is constructed it has already
 * succeeded or the process never started.
 */
func NewUserModel() *User {
	db, _ := core.Connection()

	return &User{db: db}
}

// Create inserts the user and reads back its generated id.
func (user *User) Create() error {
	query := `
        INSERT INTO users (username, password)
        VALUES ($1, $2)
        RETURNING id`

	result, err := user.db.Create(query, user.Username, user.Password)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}

	// QueryRow defers the insert error to Scan, so returning nil here hid
	// every constraint violation from the caller.
	if err := result.Scan(&user.ID); err != nil {
		return fmt.Errorf("scanning created user: %w", err)
	}

	return nil
}

// Find loads the user matching the set username.
func (user *User) Find() error {
	return user.FindByUsername()
}

// FindByUsername loads the user matching the set username.
func (user *User) FindByUsername() error {
	if user.Username == "" {
		return errors.New("username is required")
	}

	query := `
		SELECT id, username, password, created_at
		FROM users
		WHERE username = $1`

	rows, err := user.db.Read(query, user.Username)
	if err != nil {
		return fmt.Errorf("finding user: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt); err != nil {
			return fmt.Errorf("scanning user: %w", err)
		}

		return nil
	}

	// Without this a failure part way through iteration is indistinguishable
	// from an empty result.
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating users: %w", err)
	}

	return sql.ErrNoRows
}

// Update writes the model's current values back to its row.
func (user *User) Update() error {
	if user.ID == 0 {
		return errors.New("id is required")
	}

	query := `
        UPDATE users
        SET username = $1, password = $2
        WHERE id = $3`

	if _, err := user.db.Update(query, user.Username, user.Password, user.ID); err != nil {
		return fmt.Errorf("updating user: %w", err)
	}

	return nil
}

// Delete removes the model's row.
func (user *User) Delete() error {
	if user.ID == 0 {
		return errors.New("id is required")
	}

	query := `
        DELETE FROM users
        WHERE id = $1`

	if _, err := user.db.Delete(query, user.ID); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	return nil
}

/*
 * Paginate lists users newest first.
 *
 * @param limit Rows per page; defaults to 10 when not positive.
 * @param page  One-based page number; defaults to 1 when not positive.
 */
func (user *User) Paginate(limit, page int) ([]User, error) {
	if limit <= 0 {
		limit = 10
	}

	if page <= 0 {
		page = 1
	}

	query := `
        SELECT id, username, password, created_at
        FROM users
        ORDER BY id DESC
        LIMIT $1 OFFSET $2`

	rows, err := user.db.Read(query, limit, (page-1)*limit)
	if err != nil {
		return nil, fmt.Errorf("paginating users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0, limit)

	for rows.Next() {
		var found User

		if err := rows.Scan(&found.ID, &found.Username, &found.Password, &found.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}

		users = append(users, found)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating users: %w", err)
	}

	return users, nil
}
