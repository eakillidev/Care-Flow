package config

import (
	"fmt"
	"net/url"
	"os"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   time.Duration
}

func Load() Config {
	return Config{
		Port:        getEnv("API_PORT", "8080"),
		DatabaseURL: databaseURL(),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTExpiry:   24 * time.Hour,
	}
}

func databaseURL() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}

	user := getEnv("POSTGRES_USER", "careflow")
	password := os.Getenv("POSTGRES_PASSWORD")
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	database := getEnv("POSTGRES_DB", "careflow")
	sslMode := getEnv("POSTGRES_SSLMODE", "disable")

	connectionURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   database,
	}
	query := connectionURL.Query()
	query.Set("sslmode", sslMode)
	connectionURL.RawQuery = query.Encode()

	return connectionURL.String()
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
