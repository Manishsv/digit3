package config

import (
	"log"
	"os"
	"strconv"
	"strings"
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

	// Template config configuration
	TemplateConfigHost string
	TemplateConfigPath string

	//Filestore configuration
	FilestoreHost string
	FilestorePath string

	// Email configuration
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPFromAddress string
	SMTPFromName    string

	//SMS configuration
	SMSProviderURL         string
	SMSProviderUsername    string
	SMSProviderPassword    string
	SMSProviderContentType string

	//Message broker configuration
	MessageBrokerEnabled bool
	MessageBrokerType    string
	KafkaBrokers         []string
	KafkaConsumerGroup   string
	RedisAddr            string
	RedisPassword        string
	RedisDB              int
	EmailTopic           string
	SMSTopic             string
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
		ServerContextPath: getEnv("SERVER_CONTEXT_PATH", "/notification"),

		// Database configuration
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "notification_db"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		//Template config configuration
		TemplateConfigHost: getEnv("TEMPLATE_CONFIG_HOST", "http://localhost:8080"),
		TemplateConfigPath: getEnv("TEMPLATE_CONFIG_PATH", "/template-config/v1/render"),

		//Filestore configuration
		FilestoreHost: getEnv("FILESTORE_HOST", "http://localhost:8080"),
		FilestorePath: getEnv("FILESTORE_PATH", "/filestore/v1/upload"),

		//Email configuration
		SMTPHost:        getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:        getEnvAsInt("SMTP_PORT", 587),
		SMTPUsername:    getEnv("SMTP_USERNAME", "username"),
		SMTPPassword:    getEnv("SMTP_PASSWORD", "password"),
		SMTPFromAddress: getEnv("SMTP_FROM_ADDRESS", "notification@example.com"),
		SMTPFromName:    getEnv("SMTP_FROM_NAME", "Notification Service"),

		//SMS configuration
		SMSProviderURL:         getEnv("SMS_PROVIDER_URL", "https://smscountry.com/api/v3/sendsms/plain"),
		SMSProviderUsername:    getEnv("SMS_PROVIDER_USERNAME", "username"),
		SMSProviderPassword:    getEnv("SMS_PROVIDER_PASSWORD", "password"),
		SMSProviderContentType: getEnv("SMS_PROVIDER_CONTENT_TYPE", "application/x-www-form-urlencoded"),

		//Message broker configuration
		MessageBrokerEnabled: getEnvAsBool("MESSAGE_BROKER_ENABLED", false),
		MessageBrokerType:    getEnv("MESSAGE_BROKER_TYPE", "kafka"),
		KafkaBrokers:         strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaConsumerGroup:   getEnv("KAFKA_CONSUMER_GROUP", "notification-consumer-group"),
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        getEnv("REDIS_PASSWORD", ""),
		RedisDB:              getEnvAsInt("REDIS_DB", 0),
		EmailTopic:           getEnv("EMAIL_TOPIC", "notification-email"),
		SMSTopic:             getEnv("SMS_TOPIC", "notification-sms"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
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
