package repository

import (
	"context"
	"database/sql"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type AuthSessionRepository struct {
	db *bun.DB
}

func NewAuthSessionRepository(db *bun.DB) *AuthSessionRepository {
	return &AuthSessionRepository{db: db}
}

func (r *AuthSessionRepository) Create(ctx context.Context, session *model.AuthSession) error {
	_, err := r.db.NewInsert().
		Model(session).
		Column("user_id", "refresh_token", "ip_address", "user_agent", "location", "latitude", "longitude", "geocoding_source", "jkt", "expires_at").
		Returning("id, revoked, created_at").
		Exec(ctx)
	return err
}

func (r *AuthSessionRepository) FindByToken(ctx context.Context, token, userID string) (*model.AuthSession, error) {
	t := new(model.AuthSession)
	err := r.db.NewSelect().
		Model(t).
		Column("revoked", "created_at", "expires_at", "jkt").
		Where("refresh_token = ? AND user_id = ?", token, userID).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	
	return t, nil
}

func (r *AuthSessionRepository) Revoke(ctx context.Context, token, userID string) error {
	_, err := r.db.NewUpdate().
		Model((*model.AuthSession)(nil)).
		Set("revoked = true").
		Where("refresh_token = ? AND user_id = ?", token, userID).
		Exec(ctx)
	return err
}
