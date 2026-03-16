package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string
	Email  string
}

type JWTProvider struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTProvider(secret string, ttl time.Duration) *JWTProvider {
	return &JWTProvider{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (p *JWTProvider) Generate(userID, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"exp":   time.Now().Add(p.ttl).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(p.secret)
}

func (p *JWTProvider) Parse(tokenString string) (*Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return p.secret, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	sub, err := mapClaims.GetSubject()
	if err != nil || sub == "" {
		return nil, errors.New("invalid token subject")
	}

	email, ok := mapClaims["email"].(string)
	if !ok || email == "" {
		return nil, errors.New("invalid token email")
	}

	return &Claims{UserID: sub, Email: email}, nil
}
