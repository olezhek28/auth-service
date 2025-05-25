package models

import (
	"time"

	"github.com/google/uuid"
)

// User представляет пользователя в системе
type User struct {
	ID           int64     `db:"id"`
	UUID         uuid.UUID `db:"uuid"`
	Email        string    `db:"email"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// CreateUserRequest запрос на создание пользователя
type CreateUserRequest struct {
	Email    string
	Username string
	Password string
}
