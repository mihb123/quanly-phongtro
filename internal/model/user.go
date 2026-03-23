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
	ID           string
	Email        string
	PasswordHash string
	Role         Role
	FullName     string
	Phone        string
	IsActivated  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
}
