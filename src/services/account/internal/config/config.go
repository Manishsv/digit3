package config

import (
	"os"
	"strconv"
)

type KeycloakConfig struct {
	BaseURL                   string
	AdminUser                 string
	AdminPass                 string
	RealmConfigPath           string
	CitizenBrokerClientId     string
	CitizenBrokerClientSecret string
}

type NotificationConfig struct {
	BaseURL string
}

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Keycloak     KeycloakConfig
	Notification NotificationConfig
}

type ServerConfig struct {
	Port        string
	ContextPath string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:        getEnv("SERVER_PORT", "8080"),
			ContextPath: getEnv("CONTEXT_PATH", "/account"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "1234"),
			DBName:   getEnv("DB_NAME", "keycloak"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Keycloak: KeycloakConfig{
			BaseURL:                   getEnv("KEYCLOAK_BASE_URL", "https://digit-lts.digit.org/keycloak"),
			AdminUser:                 getEnv("KEYCLOAK_ADMIN_USER", "digit"),
			AdminPass:                 getEnv("KEYCLOAK_ADMIN_PASS", "digit@321"),
			RealmConfigPath:           getEnv("KEYCLOAK_REALM_CONFIG_PATH", ""),
			CitizenBrokerClientId:     getEnv("CITIZEN_BROKER_CLIENT_ID", "citizen-broker"),
			CitizenBrokerClientSecret: getEnv("CITIZEN_BROKER_CLIENT_SECRET", "zjlrLxJCFpsRIemN9pJpUC9Wy9gjWS7m"),
		},
		Notification: NotificationConfig{
			BaseURL: getEnv("NOTIFICATION_BASE_URL", "http://localhost:8080"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
