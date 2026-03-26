package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/mihb123/quanly-phongtro/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	const query = `
		SELECT id, email, password_hash, role, full_name, phone, is_activated, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	return r.getUser(ctx, query, email)
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	const query = `
		SELECT id, email, password_hash, role, full_name, phone, is_activated, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	return r.getUser(ctx, query, id)
}

func (r *UserRepository) getUser(ctx context.Context, query string, arg any) (*model.User, error) {
	var u model.User
	var role string
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&role,
		&u.FullName,
		&u.Phone,
		&u.IsActivated,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}

		return nil, fmt.Errorf("get user: %w", err)
	}

	u.Role = model.Role(role)
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	const query = `
		INSERT INTO users (email, password_hash, role, full_name, phone, is_activated)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query, u.Email, u.PasswordHash, string(u.Role), u.FullName, u.Phone, u.IsActivated).Scan(
		&u.ID,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "users_email") {
			return model.ErrAlreadyExists
		}

		return fmt.Errorf("create user: %w", err)
	}

	return nil
}
