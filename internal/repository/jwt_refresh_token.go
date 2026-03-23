package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type JWTRefreshTokenRepository struct {
	db *sql.DB
}

func NewJWTRefreshTokenRepository(db *sql.DB) *JWTRefreshTokenRepository {
	return &JWTRefreshTokenRepository{db: db}
}

func (r *JWTRefreshTokenRepository) Create(ctx context.Context, token *model.JWTRefreshToken) error {
	query := `INSERT INTO jwt_refresh_tokens (user_id, refresh_token) 
			VALUES ($1 , $2) RETURNING id, revoked, created_at`
	err := r.db.QueryRowContext(ctx, query, token.UserID, token.RefreshToken).Scan(&token.ID, &token.Revoked, &token.CreatedAt)
	return err
}

func (r *JWTRefreshTokenRepository) FindByToken(ctx context.Context, token, userID string) (bool, error) {
	query := `SELECT revoked, created_at FROM jwt_refresh_tokens WHERE refresh_token = $1 AND user_id = $2`
	var revoked bool
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query, token, userID).Scan(&revoked, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if revoked {
		return true, nil
	}
	if time.Since(createdAt) > 24*30*time.Hour {
		return true, nil
	}

	return false, nil
}

func (r *JWTRefreshTokenRepository) Revoke(ctx context.Context, token, userID string) error {
	query := `UPDATE jwt_refresh_tokens SET revoked = true WHERE refresh_token = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, token, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return nil
	}
	return nil
}
