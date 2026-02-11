package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server configuration
	HTTPPort          string
	ServerContextPath string

	// Database configuration
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
}

func Load() *Config {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading it, relying on system env vars.")
	}

	return &Config{
		// Server configuration
		HTTPPort:          getEnv("HTTP_PORT", "8080"),
		ServerContextPath: getEnv("SERVER_CONTEXT_PATH", "/idgen"),

		// Database configuration
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "idgen_db"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}
}

func getEnv(key string, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
