package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type UserRepository struct {
	db *bun.DB
}

func NewUserRepository(db *bun.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	var role string
	err := r.db.NewSelect().
		Model((*model.User)(nil)).
		Column("id", "email", "role", "full_name", "phone", "is_activated", "theme", "color_mode", "zalo_bot_token", "zalo_webhook_secret", "is_zalo_bot_active", "zalo_user_id", "created_at", "updated_at").
		Where("email = ?", email).
		Scan(ctx, &u.ID, &u.Email, &role, &u.FullName, &u.Phone, &u.IsActivated, &u.Theme, &u.ColorMode, &u.ZaloBotToken, &u.ZaloWebhookSecret, &u.IsZaloBotActive, &u.ZaloUserID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	u.Role = model.Role(role)
	return &u, nil
}

func (r *UserRepository) GetAuthUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	var role string
	err := r.db.NewSelect().
		Model((*model.User)(nil)).
		Column("id", "email", "password_hash", "role", "full_name", "phone", "is_activated", "theme", "color_mode", "zalo_bot_token", "zalo_webhook_secret", "is_zalo_bot_active", "zalo_user_id", "created_at", "updated_at").
		Where("email = ?", email).
		Scan(ctx, &u.ID, &u.Email, &u.PasswordHash, &role, &u.FullName, &u.Phone, &u.IsActivated, &u.Theme, &u.ColorMode, &u.ZaloBotToken, &u.ZaloWebhookSecret, &u.IsZaloBotActive, &u.ZaloUserID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get auth user: %w", err)
	}

	u.Role = model.Role(role)
	return &u, nil
}

func (r *UserRepository) GetByUserID(ctx context.Context, userID string) (*model.User, error) {
	var u model.User
	var role string
	err := r.db.NewSelect().
		Model((*model.User)(nil)).
		Column("id", "email", "password_hash", "role", "full_name", "phone", "is_activated", "theme", "color_mode", "zalo_bot_token", "zalo_webhook_secret", "is_zalo_bot_active", "zalo_user_id", "created_at", "updated_at").
		Where("id = ?", userID).
		Scan(ctx, &u.ID, &u.Email, &u.PasswordHash, &role, &u.FullName, &u.Phone, &u.IsActivated, &u.Theme, &u.ColorMode, &u.ZaloBotToken, &u.ZaloWebhookSecret, &u.IsZaloBotActive, &u.ZaloUserID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	u.Role = model.Role(role)
	return &u, nil
}

func (r *UserRepository) GetByPhone(ctx context.Context, phone string) (*model.User, error) {
	var u model.User
	var role string
	err := r.db.NewSelect().
		Model((*model.User)(nil)).
		Column("id", "email", "role", "full_name", "phone", "is_activated", "theme", "color_mode", "zalo_bot_token", "zalo_webhook_secret", "is_zalo_bot_active", "zalo_user_id", "created_at", "updated_at").
		Where("phone = ?", phone).
		Scan(ctx, &u.ID, &u.Email, &role, &u.FullName, &u.Phone, &u.IsActivated, &u.Theme, &u.ColorMode, &u.ZaloBotToken, &u.ZaloWebhookSecret, &u.IsZaloBotActive, &u.ZaloUserID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get user by phone: %w", err)
	}

	u.Role = model.Role(role)
	return &u, nil
}

func (r *UserRepository) GetByZaloUserID(ctx context.Context, zaloUserID string) (*model.User, error) {
	var u model.User
	var role string
	err := r.db.NewSelect().
		Model((*model.User)(nil)).
		Column("id", "email", "role", "full_name", "phone", "is_activated", "theme", "color_mode", "zalo_bot_token", "zalo_webhook_secret", "is_zalo_bot_active", "zalo_user_id", "created_at", "updated_at").
		Where("zalo_user_id = ?", zaloUserID).
		Scan(ctx, &u.ID, &u.Email, &role, &u.FullName, &u.Phone, &u.IsActivated, &u.Theme, &u.ColorMode, &u.ZaloBotToken, &u.ZaloWebhookSecret, &u.IsZaloBotActive, &u.ZaloUserID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get user by zalo user id: %w", err)
	}

	u.Role = model.Role(role)
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	_, err := r.db.NewInsert().
		Model(u).
		Column("email", "password_hash", "role", "full_name", "phone", "is_activated").
		Returning("id, created_at, updated_at, theme, color_mode").
		Exec(ctx)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "users_email") {
			return model.ErrAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepository) ActivateUser(ctx context.Context, email string) error {
	_, err := r.db.NewUpdate().
		Model((*model.User)(nil)).
		Set("is_activated = true").
		Set("updated_at = NOW()").
		Where("email = ?", email).
		Exec(ctx)
	return err
}

func (r *UserRepository) DeactivateUser(ctx context.Context, userID string) error {
	res, err := r.db.NewUpdate().
		Model((*model.User)(nil)).
		Set("is_activated = false").
		Where("id = ?", userID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("deactivate user: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("deactivate user rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, userID string, input model.UpdateUserInput) (*model.User, error) {
	q := r.db.NewUpdate().
		Model((*model.User)(nil)).
		Where("id = ?", userID).
		Returning("id, email, role, full_name, phone, is_activated, theme, color_mode, zalo_bot_token, zalo_webhook_secret, is_zalo_bot_active, zalo_user_id, created_at, updated_at")

	updated := false
	if input.FullName != nil {
		q.Set("full_name = ?", *input.FullName)
		updated = true
	}
	if input.Phone != nil {
		q.Set("phone = ?", *input.Phone)
		updated = true
	}
	if input.Theme != nil {
		q.Set("theme = ?", *input.Theme)
		updated = true
	}
	if input.ColorMode != nil {
		q.Set("color_mode = ?", *input.ColorMode)
		updated = true
	}
	if input.Email != nil {
		q.Set("email = ?", *input.Email)
		updated = true
	}
	if input.PasswordHash != nil {
		q.Set("password_hash = ?", *input.PasswordHash)
		updated = true
	}
	if input.ZaloBotToken != nil {
		q.Set("zalo_bot_token = ?", *input.ZaloBotToken)
		updated = true
	}
	if input.ZaloWebhookSecret != nil {
		q.Set("zalo_webhook_secret = ?", *input.ZaloWebhookSecret)
		updated = true
	}
	if input.IsZaloBotActive != nil {
		q.Set("is_zalo_bot_active = ?", *input.IsZaloBotActive)
		updated = true
	}
	if input.ZaloUserID != nil {
		q.Set("zalo_user_id = ?", *input.ZaloUserID)
		updated = true
	}
	if !updated {
		return r.GetByUserID(ctx, userID)
	}
	q.Set("updated_at = NOW()")

	var u model.User
	var role string
	err := q.Scan(ctx, &u.ID, &u.Email, &role, &u.FullName, &u.Phone, &u.IsActivated, &u.Theme, &u.ColorMode, &u.ZaloBotToken, &u.ZaloWebhookSecret, &u.IsZaloBotActive, &u.ZaloUserID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}

	u.Role = model.Role(role)
	return &u, nil
}

// GetAllUsersWithZaloToken returns all users who have a configured Zalo bot token.
// Used by the cron job to periodically verify token validity.
func (r *UserRepository) GetAllUsersWithZaloToken(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.NewSelect().
		Model((*model.User)(nil)).
		Column("id", "email", "zalo_bot_token", "is_zalo_bot_active").
		Where("zalo_bot_token IS NOT NULL AND zalo_bot_token != ''").
		Rows(ctx)
	if err != nil {
		return nil, fmt.Errorf("get users with zalo token: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Email, &u.ZaloBotToken, &u.IsZaloBotActive); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
