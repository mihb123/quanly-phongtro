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
	AppEnv                string
	AppURL                string
	AppURLDev             string
	PostgresDSN           string
	AccessTokenJWTSecret  string
	RefreshTokenJWTSecret string
	TokenTTL              time.Duration
	SMTPHost              string
	SMTPPort              string
	SMTPUsername          string
	SMTPPassword          string
	MailFromEmail         string
	MailFromName          string
	OTPEXpireMinutes      time.Duration
	ZaloBotEncryptionKey  string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	appPort := getOrDefault("APP_PORT", "8080")
	appEnv := getOrDefault("APP_ENV", "prod")
	appURL := getOrDefault("APP_URL", "localhost:8080")
	appURLDev := os.Getenv("APP_URL_DEV")
	postgresDSN := os.Getenv("POSTGRES_DSN")
	accessTokenJWTSecret := os.Getenv("ACCESS_TOKEN_JWT_SECRET")
	refreshTokenJWTSecret := os.Getenv("REFRESH_TOKEN_JWT_SECRET")
	ttlMinutes := getOrDefault("JWT_TTL_MINUTES", "60")
	smtpHost := getOrDefault("SMTP_HOST", "smtp.gmail.com")
	smtpPort := getOrDefault("SMTP_PORT", "587")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	mailFromEmail := getOrDefault("MAIL_FROM_EMAIL", "no-reply@example.com")
	mailFromName := getOrDefault("MAIL_FROM_NAME", "Go App")
	otpExpireMinutes := os.Getenv("OTP_EXPIRE_MINIUTES")
	zaloBotEncryptionKey := os.Getenv("ZALO_BOT_ENCRYPTION_KEY")

	if smtpUsername == "" {
		return nil, errors.New("SMTP_USERNAME is required")
	}

	if smtpPassword == "" {
		return nil, errors.New("SMTP_PASSWORD is required")
	}

	if zaloBotEncryptionKey == "" {
		return nil, errors.New("ZALO_BOT_ENCRYPTION_KEY is required")
	}

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

	otpMinutes, err := strconv.Atoi(otpExpireMinutes)
	if err != nil || otpMinutes <= 0 {
		return nil, errors.New("JWT_TTL_MINUTES must be a positive integer")
	}

	return &Config{
		AppPort:               appPort,
		AppEnv:                appEnv,
		AppURL:                appURL,
		AppURLDev:             appURLDev,
		PostgresDSN:           postgresDSN,
		AccessTokenJWTSecret:  accessTokenJWTSecret,
		RefreshTokenJWTSecret: refreshTokenJWTSecret,
		TokenTTL:              time.Duration(minutes) * time.Minute,
		OTPEXpireMinutes:      time.Duration(otpMinutes) * time.Minute,
		SMTPHost:              smtpHost,
		SMTPPort:              smtpPort,
		SMTPUsername:          smtpUsername,
		SMTPPassword:          smtpPassword,
		MailFromEmail:         mailFromEmail,
		MailFromName:          mailFromName,
		ZaloBotEncryptionKey:  zaloBotEncryptionKey,
	}, nil
}

func getOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
