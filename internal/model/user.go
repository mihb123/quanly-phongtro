package model

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrAlreadyExists = errors.New("user already exists")
)

type Role string

const (
	RoleManager Role = "MANAGER"
	RoleTenant  Role = "TENANT"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	FullName     string    `json:"full_name"`
	Phone        string    `json:"phone"`
	IsActivated  bool      `json:"is_activated"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
	ActivateUser(ctx context.Context, email string) error
	GetByUserID(ctx context.Context, userID string) (*User, error)
	UpdateUser(ctx context.Context, userID string, input UpdateUserInput) (*User, error)
	DeactivateUser(ctx context.Context, userID string) error
}

// UpdateUserInput holds optional fields for a partial user update.
// Only non-nil pointer fields will be written to the DB.
type UpdateUserInput struct {
	FullName *string
	Phone    *string
	Email    *string
}
