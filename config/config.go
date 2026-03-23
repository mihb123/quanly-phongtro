package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort               string
	PostgresDSN           string
	AccessTokenJWTSecret  string
	RefreshTokenJWTSecret string
	TokenTTL              time.Duration
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	appPort := getOrDefault("APP_PORT", "8080")
	postgresDSN := os.Getenv("POSTGRES_DSN")
	accessTokenJWTSecret := os.Getenv("ACCESS_TOKEN_JWT_SECRET")
	refreshTokenJWTSecret := os.Getenv("REFRESH_TOKEN_JWT_SECRET")
	ttlMinutes := getOrDefault("JWT_TTL_MINUTES", "60")

	if postgresDSN == "" {
		return nil, errors.New("POSTGRES_DSN is required")
	}

	if accessTokenJWTSecret == "" {
		return nil, errors.New("ACCESS_TOKEN_JWT_SECRET is required")
	}

	if refreshTokenJWTSecret == "" {
		return nil, errors.New("REFRESH_TOKEN_JWT_SECRET is required")
	}

	minutes, err := strconv.Atoi(ttlMinutes)
	if err != nil || minutes <= 0 {
		return nil, errors.New("JWT_TTL_MINUTES must be a positive integer")
	}

	return &Config{
		AppPort:               appPort,
		PostgresDSN:           postgresDSN,
		AccessTokenJWTSecret:  accessTokenJWTSecret,
		RefreshTokenJWTSecret: refreshTokenJWTSecret,
		TokenTTL:              time.Duration(minutes) * time.Minute,
	}, nil
}

func getOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
