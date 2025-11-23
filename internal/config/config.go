package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	APIPort           string
	AuthAPIAllowedIPs []string
	DBAPIAllowedIPs   []string
	LogFile           string
	LogMaxSize        int
	LogMaxBackups     int
	LogMaxAge         int
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists, but don't fail if it doesn't (might be env vars)
	_ = godotenv.Load()

	cfg := &Config{
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "akaserver"),
		DBPassword:        getEnv("DB_PASSWORD", "akaserver"),
		DBName:            getEnv("DB_NAME", "akaserverdb"),
		APIPort:           getEnv("API_PORT", "8080"),
		AuthAPIAllowedIPs: getEnvAsSlice("AUTH_API_ALLOWED_IPS"),
		DBAPIAllowedIPs:   getEnvAsSlice("DB_API_ALLOWED_IPS"),
		LogFile:           getEnv("LOG_FILE", "akaserver.log"),
		LogMaxSize:        getEnvAsInt("LOG_MAX_SIZE", 10),
		LogMaxBackups:     getEnvAsInt("LOG_MAX_BACKUPS", 3),
		LogMaxAge:         getEnvAsInt("LOG_MAX_AGE", 28),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsSlice(key string) []string {
	value := getEnv(key, "")
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return fallback
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallback
	}
	return value
}
