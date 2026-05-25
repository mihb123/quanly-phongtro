package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type JWTRefreshTokenRepository struct {
	db *bun.DB
}

func NewJWTRefreshTokenRepository(db *bun.DB) *JWTRefreshTokenRepository {
	return &JWTRefreshTokenRepository{db: db}
}

func (r *JWTRefreshTokenRepository) Create(ctx context.Context, token *model.JWTRefreshToken) error {
	_, err := r.db.NewInsert().
		Model(token).
		Column("user_id", "refresh_token").
		Returning("id, revoked, created_at").
		Exec(ctx)
	return err
}

func (r *JWTRefreshTokenRepository) FindByToken(ctx context.Context, token, userID string) (bool, error) {
	t := new(model.JWTRefreshToken)
	err := r.db.NewSelect().
		Model(t).
		Column("revoked", "created_at").
		Where("refresh_token = ? AND user_id = ?", token, userID).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if t.Revoked {
		return true, nil
	}
	if time.Since(t.CreatedAt) > 24*30*time.Hour {
		return true, nil
	}

	return false, nil
}

func (r *JWTRefreshTokenRepository) Revoke(ctx context.Context, token, userID string) error {
	_, err := r.db.NewUpdate().
		Model((*model.JWTRefreshToken)(nil)).
		Set("revoked = true").
		Where("refresh_token = ? AND user_id = ?", token, userID).
		Exec(ctx)
	return err
}
