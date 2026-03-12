package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Host            string
	Port            string
	AppEnv          string
	PostgresDSN     string
	DBMaxOpenConns  int
	DBMaxIdleConns  int
	DBConnMaxLifeM  int
	DBConnMaxIdleM  int
	ReadTimeoutSec  int
	WriteTimeoutSec int
	IdleTimeoutSec  int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Host:            getEnv("HOST", "0.0.0.0"),
		Port:            getEnv("PORT", "8080"),
		AppEnv:          getEnv("APP_ENV", "development"),
		PostgresDSN:     getEnv("POSTGRES_DSN", ""),
		DBMaxOpenConns:  getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:  getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifeM:  getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 30),
		DBConnMaxIdleM:  getEnvAsInt("DB_CONN_MAX_IDLE_MINUTES", 5),
		ReadTimeoutSec:  getEnvAsInt("HTTP_READ_TIMEOUT_SEC", 10),
		WriteTimeoutSec: getEnvAsInt("HTTP_WRITE_TIMEOUT_SEC", 30),
		IdleTimeoutSec:  getEnvAsInt("HTTP_IDLE_TIMEOUT_SEC", 60),
	}

	if cfg.Port == "" {
		return nil, fmt.Errorf("PORT must be set")
	}
	if cfg.PostgresDSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN must be set")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}
