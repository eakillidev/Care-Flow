package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	JWTExpiry         time.Duration
	EVVGeofenceMeters float64
	EVVTimeTolerance  time.Duration
}

func Load() (Config, error) {
	geofenceMeters, err := positiveFloat("EVV_GEOFENCE_METERS", 200)
	if err != nil {
		return Config{}, err
	}
	toleranceMinutes, err := positiveFloat("EVV_TIME_TOLERANCE_MINUTES", 15)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Port:              getEnv("API_PORT", "8080"),
		DatabaseURL:       databaseURL(),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		JWTExpiry:         24 * time.Hour,
		EVVGeofenceMeters: geofenceMeters,
		EVVTimeTolerance:  time.Duration(toleranceMinutes * float64(time.Minute)),
	}, nil
}

func positiveFloat(key string, fallback float64) (float64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive number", key)
	}
	return parsed, nil
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
