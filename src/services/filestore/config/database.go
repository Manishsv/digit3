package config

import (
	"fmt"
	"os"
)

// onfig holds all database configuration parameters
type Config struct {
	Host                 string
	Port                 string
	User                 string
	Password             string
	Name                 string
	SSLMode              string
	ServerContextPath    string
	MinioEndpoint        string
	MinioAccessKey       string
	MinioSecretKey       string
	MinioWriteBucketName string
	MinioReadBucketName  string
	MinioUseSSL          bool
}

// NewConfig creates a new Config instance with values from environment variables
func NewConfig() *Config {
	return &Config{
		Host:                 getEnvOrDefault("DB_HOST", "minio.default"),
		Port:                 getEnvOrDefault("DB_PORT", "5432"),
		User:                 getEnvOrDefault("DB_USER", "postgres"),
		Password:             getEnvOrDefault("DB_PASSWORD", "postgres"),
		Name:                 getEnvOrDefault("DB_NAME", "postgres"),
		SSLMode:              getEnvOrDefault("DB_SSL_MODE", "disable"),
		ServerContextPath:    getEnvOrDefault("SERVER_CONTEXT_PATH", "/filestore"),
		MinioEndpoint:        getEnvOrDefault("MINIO_ENDPOINT"),
		MinioAccessKey:       getEnvOrDefault("MINIO_ACCESS_KEY"),
		MinioSecretKey:       getEnvOrDefault("MINIO_SECRET_KEY"),
		MinioWriteBucketName: getEnvOrDefault("MINIO_BUCKET"),
		MinioReadBucketName:  getEnvOrDefault("MINIO_READ_BUCKET"),
		MinioUseSSL:          getEnvOrDefault("MINIO_USE_SSL", "false") == "true",
	}
}

// GetConnectionString returns the formatted database connection string
func (c *Config) GetConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode)
}

// getEnvOrDefault returns the value of the environment variable or a default value if not set
func getEnvOrDefault(key string, defaultValue ...string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return "" // Return empty string if no default value is provided
}
