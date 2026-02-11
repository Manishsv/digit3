package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server ServerConfig
	DB     DBConfig
}

type ServerConfig struct {
	Port        string
	ContextPath string
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

func LoadConfig() (*Config, error) {
	// Try to load .env file (optional)
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5433"))
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			Port:        getEnv("SERVER_PORT", "8080"),
			ContextPath: getEnv("CONTEXT_PATH", "/workflow"),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "postgres"),
			SSLMode:  getEnvFirst([]string{"DB_SSLMODE", "DB_SSL_MODE"}, "disable"),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFirst(keys []string, defaultValue string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return defaultValue
}
