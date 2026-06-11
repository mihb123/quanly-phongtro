package model

import (
	"context"
	"time"
)

type AuthSession struct {
	ID           string    `bun:",pk,type:uuid,default:gen_random_uuid()"`
	UserID       string    `bun:"user_id"`
	RefreshToken string    `bun:"refresh_token"`
	Revoked      bool      `bun:"revoked"`
	IPAddress    string    `bun:"ip_address"`
	UserAgent    string    `bun:"user_agent"`
	Location        string    `bun:"location"`
	Latitude        *float64  `bun:"latitude"`
	Longitude       *float64  `bun:"longitude"`
	GeocodingSource *string   `bun:"geocoding_source"`
	JKT             string    `bun:"jkt"`
	CreatedAt    time.Time `bun:"created_at"`
	ExpiresAt    time.Time `bun:"expires_at"`
}

type AuthSessionRepository interface {
	Create(ctx context.Context, session *AuthSession) error
	FindByToken(ctx context.Context, token, userID string) (*AuthSession, error)
	Revoke(ctx context.Context, token, userID string) error
}
