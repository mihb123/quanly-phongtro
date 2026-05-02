package security

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
)

type Claims struct {
	IsActivated bool   `json:"is_activated"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	jwt.RegisteredClaims
}

type JWTProvider struct {
	accessSecret  []byte
	refreshSecret []byte
	ttl           time.Duration
	jwtRepo       model.JWTRefreshTokenRepository
}

func NewJWTProvider(accessSecret, refreshSecret string, ttl time.Duration, jwtRepo model.JWTRefreshTokenRepository) *JWTProvider {
	return &JWTProvider{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		ttl:           ttl,
		jwtRepo:       jwtRepo,
	}
}

func (p *JWTProvider) GenerateAccessToken(role, email, userID string, isActivated bool) (string, error) {
	claims := Claims{
		IsActivated: isActivated,
		Email:       email,
		Role:        role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(p.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(p.accessSecret)
}

func (p *JWTProvider) GenerateRefreshToken(ctx context.Context, userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(p.ttl * 24 * 30).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(p.refreshSecret)
	if err != nil {
		return "", err
	}

	refreshToken := &model.JWTRefreshToken{
		UserID:       userID,
		RefreshToken: signedToken,
	}
	if err := p.jwtRepo.Create(ctx, refreshToken); err != nil {
		return "", err
	}

	return signedToken, nil
}

func (p *JWTProvider) RevokeRefreshToken(ctx context.Context, token string, userID string) error {
	return p.jwtRepo.Revoke(ctx, token, userID)
}

func (p *JWTProvider) FindByToken(ctx context.Context, token string, userID string) (bool, error) {
	return p.jwtRepo.FindByToken(ctx, token, userID)
}

func (p *JWTProvider) GetAccessTokenTTL() time.Duration {
	return p.ttl
}

func (p *JWTProvider) Parse(tokenString string, tokenType string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		switch tokenType {
		case "access":
			return p.accessSecret, nil
		case "refresh":
			return p.refreshSecret, nil
		default:
			return nil, errors.New("invalid token type")
		}
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
