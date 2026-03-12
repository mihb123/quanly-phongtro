package repository

import (
	"context"
	"database/sql"

	"github.com/mihb123/quanly-phongtro/src/Model"
)

type UserRepository interface {
	Create(requestContext context.Context, email, fullName, passwordHash string) error
	FindByEmail(requestContext context.Context, email string) (*model.User, error)
}

type userRepository struct {
	databaseConnection *sql.DB
}

func NewUserRepository(databaseConnection *sql.DB) UserRepository {
	return &userRepository{databaseConnection: databaseConnection}
}

func (repository *userRepository) Create(requestContext context.Context, email, fullName, passwordHash string) error {
	const query = `
		INSERT INTO users (email, full_name, password_hash, role)
		VALUES ($1, $2, $3, 'MANAGER')
	`
	_, err := repository.databaseConnection.ExecContext(requestContext, query, email, fullName, passwordHash)
	return err
}

func (repository *userRepository) FindByEmail(requestContext context.Context, email string) (*model.User, error) {
	const query = `
		SELECT id::text, email, password_hash, role, COALESCE(full_name, ''), COALESCE(phone, ''), is_activated, created_at, updated_at
		FROM users
		WHERE email = $1
		LIMIT 1
	`
	foundUser := &model.User{}
	err := repository.databaseConnection.QueryRowContext(requestContext, query, email).Scan(
		&foundUser.ID, &foundUser.Email, &foundUser.PasswordHash, &foundUser.Role,
		&foundUser.FullName, &foundUser.Phone, &foundUser.IsActivated,
		&foundUser.CreatedAt, &foundUser.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return foundUser, nil
}
