package model

import (
	"context"
	"time"
)

type JWTRefreshToken struct {
	ID           string
	UserID       string
	RefreshToken string
	Revoked      bool
	CreatedAt    time.Time
}

type JWTRefreshTokenRepository interface {
	Create(ctx context.Context, token *JWTRefreshToken) error
	FindByToken(ctx context.Context, token, userID string) (bool, error)
	Revoke(ctx context.Context, token, userID string) error
}
