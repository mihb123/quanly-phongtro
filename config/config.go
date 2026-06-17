package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort                string
	AppEnv                 string
	AppURL                 string
	AppURLDev              string
	PostgresDSN            string
	AccessTokenJWTSecret   string
	RefreshTokenJWTSecret  string
	TokenTTL               time.Duration
	SMTPHost               string
	SMTPPort               string
	SMTPUsername           string
	SMTPPassword           string
	MailFromEmail          string
	MailFromName           string
	OTPEXpireMinutes       time.Duration
	ZaloBotEncryptionKey   string
	AppSecretEncryptionKey string
	GoogleMapAPIKey        string
	PayOSClientID          string
	PayOSApiKey            string
	PayOSChecksumKey       string
	CookieSecure           bool
	TrustedProxyCIDRs      []netip.Prefix
	UploadURLSigningKey    string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	appPort := getOrDefault("APP_PORT", "8080")
	appEnv := getOrDefault("APP_ENV", "prod")
	appURL := getOrDefault("APP_URL", "localhost:8080")
	appURLDev := os.Getenv("APP_URL_DEV")
	cookieSecure, err := getCookieSecure(appEnv)
	if err != nil {
		return nil, err
	}
	trustedProxyCIDRs, err := parseTrustedProxyCIDRs(os.Getenv("TRUSTED_PROXIES"))
	if err != nil {
		return nil, err
	}
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
	otpExpireMinutes := os.Getenv("OTP_EXPIRE_MINUTES")
	if otpExpireMinutes == "" {
		otpExpireMinutes = os.Getenv("OTP_EXPIRE_MINIUTES")
	}
	zaloBotEncryptionKey := os.Getenv("ZALO_BOT_ENCRYPTION_KEY")
	appSecretEncryptionKey := getOrDefault("APP_SECRET_ENCRYPTION_KEY", zaloBotEncryptionKey)
	googleMapAPIKey := os.Getenv("GOOGLE_MAP_API_KEY")

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
		return nil, errors.New("OTP_EXPIRE_MINUTES must be a positive integer")
	}

	payosClientID := os.Getenv("PAYOS_CLIENT_ID")
	payosAPIKey := os.Getenv("PAYOS_API_KEY")
	payosChecksumKey := os.Getenv("PAYOS_CHECKSUM_KEY")
	uploadURLSigningKey := getOrDefault("UPLOAD_URL_SIGNING_KEY", accessTokenJWTSecret)

	return &Config{
		AppPort:                appPort,
		AppEnv:                 appEnv,
		AppURL:                 appURL,
		AppURLDev:              appURLDev,
		PostgresDSN:            postgresDSN,
		AccessTokenJWTSecret:   accessTokenJWTSecret,
		RefreshTokenJWTSecret:  refreshTokenJWTSecret,
		TokenTTL:               time.Duration(minutes) * time.Minute,
		OTPEXpireMinutes:       time.Duration(otpMinutes) * time.Minute,
		SMTPHost:               smtpHost,
		SMTPPort:               smtpPort,
		SMTPUsername:           smtpUsername,
		SMTPPassword:           smtpPassword,
		MailFromEmail:          mailFromEmail,
		MailFromName:           mailFromName,
		ZaloBotEncryptionKey:   zaloBotEncryptionKey,
		AppSecretEncryptionKey: appSecretEncryptionKey,
		GoogleMapAPIKey:        googleMapAPIKey,
		PayOSClientID:          payosClientID,
		PayOSApiKey:            payosAPIKey,
		PayOSChecksumKey:       payosChecksumKey,
		CookieSecure:           cookieSecure,
		TrustedProxyCIDRs:      trustedProxyCIDRs,
		UploadURLSigningKey:    uploadURLSigningKey,
	}, nil
}

// getCookieSecure resolves the cookie Secure flag from env or app environment.
func getCookieSecure(appEnv string) (bool, error) {
	value := os.Getenv("COOKIE_SECURE")
	if value == "" {
		return appEnv != "dev", nil
	}

	secure, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("COOKIE_SECURE must be a boolean: %w", err)
	}
	return secure, nil
}

// parseTrustedProxyCIDRs parses comma-separated proxy CIDRs or single IPs.
func parseTrustedProxyCIDRs(value string) ([]netip.Prefix, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parts := strings.Split(value, ",")
	prefixes := make([]netip.Prefix, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		prefix, err := netip.ParsePrefix(part)
		if err != nil {
			addr, addrErr := netip.ParseAddr(part)
			if addrErr != nil {
				return nil, fmt.Errorf("invalid TRUSTED_PROXIES entry %q: %w", part, err)
			}
			prefix = netip.PrefixFrom(addr, addr.BitLen())
		}
		prefixes = append(prefixes, prefix.Masked())
	}

	return prefixes, nil
}

func getOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
