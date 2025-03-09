package models

import (
	"database/sql"
	"errors"
	"log"
	"web-app/app/services/core"
)

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	CreatedAt string `json:"created_at"`
	db        *core.PostgresService
}

func NewUserModel() *User {
	db, _ := core.NewPostgresService()
	return &User{
		db: db,
	}
}

// Create implements the Model interface Create method
func (user *User) Create() error {
	query := `
        INSERT INTO users (username, password)
        VALUES ($1, $2)
        RETURNING id`

	result, err := user.db.Create(query, user.Username, user.Password)
	if err != nil {
		log.Printf("error creating user: %v", err)
		return err
	}

	err = result.Scan(&user.ID)
	if err != nil {
		log.Printf("error scanning user: %v", err)
	}

	return nil
}

// Find implements the Model interface Find method
func (user *User) Find() error {
	if user.Username == "" {
		return errors.New("username is required")
	}

	query := `
        SELECT id, username, password, created_at 
        FROM users 
        WHERE username = $1`

	rows, err := user.db.Read(query, user.Username)
	if err != nil {
		log.Printf("error finding user: %v", err)
		return err
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt)
		if err != nil {
			log.Printf("error scanning user: %v", err)
			return err
		}
		return nil
	}

	return sql.ErrNoRows
}

// FindByUsername implements the Model interface FindByUsername method
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
		log.Printf("error finding user: %v", err)
		return err
	}
	defer rows.Close()

	if rows.Next() {
		err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt)
		if err != nil {
			log.Printf("error scanning user: %v", err)
			return err
		}
		return nil
	}

	return sql.ErrNoRows
}

// Update implements the Model interface Update method
func (user *User) Update() error {
	if user.ID == 0 {
		return errors.New("id is required")
	}

	query := `
        UPDATE users 
        SET username = $1, password = $2
        WHERE id = $3`

	_, err := user.db.Update(query, user.Username, user.Password, user.ID)
	if err != nil {
		log.Printf("error updating user: %v", err)
		return err
	}

	return nil
}

// Delete implements the Model interface Delete method
func (user *User) Delete() error {
	if user.ID == 0 {
		return errors.New("id is required")
	}

	query := `
        DELETE FROM users 
        WHERE id = $1`

	_, err := user.db.Delete(query, user.ID)
	if err != nil {
		log.Printf("error deleting user: %v", err)
		return err
	}

	return nil
}

func (user *User) Paginate(limit, page int) ([]User, error) {
	// Set default values
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit
	query := `
        SELECT id, username, password, created_at
        FROM users
        ORDER BY id DESC
        LIMIT $1 OFFSET $2`

	rows, err := user.db.Read(query, limit, offset)
	if err != nil {
		log.Printf("error paginating users: %v", err)
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0, limit) // Pre-allocate slice with capacity
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Password,
			&user.CreatedAt,
		)
		if err != nil {
			log.Printf("error scanning user: %v", err)
			return nil, err
		}
		users = append(users, user)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		log.Printf("error iterating over rows: %v", err)
		return nil, err
	}

	return users, nil
}
