package model

import "time"

type UserRole string

const (
	RoleManager UserRole = "MANAGER"
	RoleTenant  UserRole = "TENANT"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         UserRole
	FullName     string
	Phone        string
	IsActivated  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
