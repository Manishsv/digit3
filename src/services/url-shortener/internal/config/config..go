package config

import (
	"log"
	"os"
	"strconv"
	"time"

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

	// Cache configuration
	CacheEnabled  bool
	CacheType     string
	CacheTTL      time.Duration
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	ShortKeyMinLength  int
	MaxShortKeyRetries int
	HostName           string
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
		ServerContextPath: getEnv("SERVER_CONTEXT_PATH", "/url-shortener"),

		// Database configuration
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "url_shortener_db"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		// Cache configuration
		CacheEnabled:  getEnvAsBool("CACHE_ENABLED", false),
		CacheType:     getEnv("CACHE_TYPE", "redis"),
		CacheTTL:      getEnvAsDuration("CACHE_TTL", 24*time.Hour),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		ShortKeyMinLength:  getEnvAsInt("SHORT_KEY_MIN_LENGTH", 4),
		MaxShortKeyRetries: getEnvAsInt("MAX_SHORT_KEY_RETRIES", 10),
		HostName:           getEnv("HOST_NAME", "http://localhost:8080"),
	}
}

func getEnv(key string, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := os.Getenv(key)
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
