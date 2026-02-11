package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Kafka         KafkaConfig
	Filestore     FilestoreConfig
	Cache         CacheConfig
	OpenTelemetry OpenTelemetryConfig
}

// ServerConfig holds server related configuration
type ServerConfig struct {
	Port         string
	ContextPath  string
	ReadTimeout  int
	WriteTimeout int
}

// DatabaseConfig holds database related configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// KafkaConfig holds Kafka related configuration
type KafkaConfig struct {
	BootstrapServers string
	ConsumerGroupID  string
	Topics           KafkaTopics
}

type KafkaTopics struct {
	CreateBoundary             string
	UpdateBoundary             string
	CreateBoundaryHierarchy    string
	UpdateBoundaryHierarchy    string
	CreateBoundaryRelationship string
	UpdateBoundaryRelationship string
}

// FilestoreConfig holds filestore related configuration
type FilestoreConfig struct {
	BasePath string
	Endpoint string
}

// CacheConfig holds cache related configuration
type CacheConfig struct {
	Type  string
	Redis RedisConfig
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// OpenTelemetryConfig holds OpenTelemetry related configuration
type OpenTelemetryConfig struct {
	ServiceName   string
	OTLPEndpoint  string
	Protocol      string
	SamplingRatio float64
	Enabled       bool
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ContextPath:  getEnv("SERVER_CONTEXT_PATH", "/boundary"),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 30),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "1234"),
			DBName:   getEnv("DB_NAME", "bound"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Kafka: KafkaConfig{
			BootstrapServers: getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
			ConsumerGroupID:  getEnv("KAFKA_CONSUMER_GROUP_ID", "boundary-service"),
			Topics: KafkaTopics{
				CreateBoundary:             getEnv("KAFKA_TOPIC_CREATE_BOUNDARY", "create-boundary-entity"),
				UpdateBoundary:             getEnv("KAFKA_TOPIC_UPDATE_BOUNDARY", "update-boundary-entity"),
				CreateBoundaryHierarchy:    getEnv("KAFKA_TOPIC_CREATE_BOUNDARY_HIERARCHY", "save-boundary-hierarchy-definition"),
				UpdateBoundaryHierarchy:    getEnv("KAFKA_TOPIC_UPDATE_BOUNDARY_HIERARCHY", "update-boundary-hierarchy-definition"),
				CreateBoundaryRelationship: getEnv("KAFKA_TOPIC_CREATE_BOUNDARY_RELATIONSHIP", "save-boundary-relationship"),
				UpdateBoundaryRelationship: getEnv("KAFKA_TOPIC_UPDATE_BOUNDARY_RELATIONSHIP", "update-boundary-relationship"),
			},
		},
		Filestore: FilestoreConfig{
			BasePath: getEnv("FILESTORE_BASEPATH", "http://localhost:8080"),
			Endpoint: getEnv("FILESTORE_ENDPOINT", "/filestore/v1/files"),
		},
		Cache: CacheConfig{
			Type: getEnv("CACHE_TYPE", "redis"),
			Redis: RedisConfig{
				Addr:     getEnv("CACHE_REDIS_ADDR", "localhost:6379"),
				Password: getEnv("CACHE_REDIS_PASSWORD", ""),
				DB:       getEnvAsInt("CACHE_REDIS_DB", 0),
			},
		},
		OpenTelemetry: OpenTelemetryConfig{
			ServiceName:   getEnv("OTEL_SERVICE_NAME", "boundary-service"),
			OTLPEndpoint:  getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "jaeger-collector.tracing:4318"),
			SamplingRatio: getEnvAsFloat("OTEL_TRACES_SAMPLER_ARG", 1.0),
			Enabled:       getEnvAsBool("OTEL_ENABLED", true),
		},
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
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

// getEnvAsFloat gets an environment variable as a float64 or returns a default value
func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsBool gets an environment variable as a boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
