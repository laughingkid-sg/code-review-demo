package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	Port          string
	DatabaseURL   string
	RedisURL      string
	RedisPassword string
	RedisDB       int
	JWTSecret     string
	RateLimitRPM  int
	Env           string
}

// Load reads configuration from environment variables with fallback defaults.
func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8081"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/orderdb?sslmode=disable"),
		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		JWTSecret:     getEnv("JWT_SECRET", "medium-api-jwt-secret-key-98765"),
		RateLimitRPM:  getEnvInt("RATE_LIMIT_RPM", 100),
		Env:           getEnv("ENV", "development"),
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
