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
	var u model.User
	var role string
	err := r.db.QueryRowContext(ctx, query, email).Scan(
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

func (r *UserRepository) GetByUserID(ctx context.Context, userID string) (*model.User, error) {
	const query = `
		SELECT id, email, password_hash, role, full_name, phone, is_activated,
		       COALESCE(cccd_path, ''), COALESCE(identity_card, ''), COALESCE(contract_path, ''),
		       created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var u model.User
	var role string
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&role,
		&u.FullName,
		&u.Phone,
		&u.IsActivated,
		&u.CCCDPath,
		&u.IdentityCard,
		&u.ContractPath,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}

		return nil, fmt.Errorf("get user by id: %w", err)
	}

	u.Role = model.Role(role)
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	const query = `
		INSERT INTO users (email, password_hash, role, full_name, phone, is_activated, cccd_path, identity_card, contract_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query, u.Email, u.PasswordHash, string(u.Role), u.FullName, u.Phone, u.IsActivated, u.CCCDPath, u.IdentityCard, u.ContractPath).Scan(
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

func (r *UserRepository) ActivateUser(ctx context.Context, email string) error {
	query := `UPDATE users SET is_activated = true, updated_at = NOW() WHERE email = $1`
	row := r.db.QueryRowContext(ctx, query, email)
	return row.Err()
}

func (r *UserRepository) DeactivateUser(ctx context.Context, userID string) (string, string, error) {
	const query = `
		UPDATE users SET is_activated = false
		WHERE id = $1
		RETURNING COALESCE(cccd_path, ''), COALESCE(contract_path, '')
	`
	var cccdPath, contractPath string
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&cccdPath, &contractPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", model.ErrNotFound
		}
		return "", "", fmt.Errorf("deactivate user: %w", err)
	}
	return cccdPath, contractPath, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, userID string, input model.UpdateUserInput) (*model.User, error) {
	setClauses := []string{}
	args := []interface{}{}
	i := 1

	if input.FullName != nil {
		setClauses = append(setClauses, fmt.Sprintf("full_name = $%d", i))
		args = append(args, *input.FullName)
		i++
	}
	if input.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone = $%d", i))
		args = append(args, *input.Phone)
		i++
	}
	if input.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", i))
		args = append(args, *input.Email)
		i++
	}
	if input.IdentityCard != nil {
		setClauses = append(setClauses, fmt.Sprintf("identity_card = $%d", i))
		args = append(args, *input.IdentityCard)
		i++
	}
	if input.CCCDPath != nil {
		setClauses = append(setClauses, fmt.Sprintf("cccd_path = $%d", i))
		args = append(args, *input.CCCDPath)
		i++
	}
	if input.ContractPath != nil {
		setClauses = append(setClauses, fmt.Sprintf("contract_path = $%d", i))
		args = append(args, *input.ContractPath)
		i++
	}

	if len(setClauses) == 0 {
		return r.GetByUserID(ctx, userID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, userID)

	query := fmt.Sprintf(
		`UPDATE users SET %s WHERE id = $%d
		RETURNING id, email, password_hash, role, full_name, phone, is_activated, cccd_path, identity_card, contract_path, created_at, updated_at`,
		strings.Join(setClauses, ", "), i,
	)

	var u model.User
	var role string
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&role,
		&u.FullName,
		&u.Phone,
		&u.IsActivated,
		&u.CCCDPath,
		&u.IdentityCard,
		&u.ContractPath,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}

	u.Role = model.Role(role)
	return &u, nil
}
