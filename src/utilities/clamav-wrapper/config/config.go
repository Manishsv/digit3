package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// KafkaConfig holds Kafka specific configuration.
type KafkaConfig struct {
	Brokers         []string
	Topic           string
	ConsumerGroupID string
}

// RedisConfig holds Redis specific configuration.
type RedisConfig struct {
	Address  string
	Key      string
	Password string // Optional
	DB       int    // Optional, defaults to 0
}

var (
	MessageBrokerType        string
	KafkaCfg                 KafkaConfig
	RedisCfg                 RedisConfig
	ClamAVHost               string
	ClamAVPort               int
	ClamAVDialTimeoutSeconds int
	ClamAVChunkSizeKB        int
	ClamAVMaxFileSizeMB      int
	MinioEndpoint            string
	MinioAccessKey           string
	MinioSecretKey           string
	StagingBucket            string
	CleanBucket              string
	QuarantineBucket         string
	UseSSL                   bool
)

func Init() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading it, relying on system env vars.")
	}

	MessageBrokerType = getEnv("MESSAGE_BROKER_TYPE", "kafka")

	// Populate KafkaConfig
	KafkaCfg.Brokers = strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	KafkaCfg.Topic = getEnv("KAFKA_TOPIC", "file-scan-clamav")
	KafkaCfg.ConsumerGroupID = getEnv("KAFKA_CONSUMER_GROUP_ID", "filestore-antivirus-group")

	// Populate RedisConfig
	RedisCfg.Address = getEnv("REDIS_ADDRESS", "localhost:6379")
	RedisCfg.Key = getEnv("REDIS_KEY", "file-scan-clamav")
	RedisCfg.Password = getEnv("REDIS_PASSWORD", "") // Default to no password
	RedisCfg.DB = getEnvAsInt("REDIS_DB", 0)         // Default to DB 0

	ClamAVHost = getEnv("CLAMAV_HOST", "localhost")
	ClamAVPort = getEnvAsInt("CLAMAV_PORT", 3310)
	ClamAVDialTimeoutSeconds = getEnvAsInt("CLAMAV_DIAL_TIMEOUT_SECONDS", 10)
	ClamAVChunkSizeKB = getEnvAsInt("CLAMAV_CHUNK_SIZE_KB", 32)
	ClamAVMaxFileSizeMB = getEnvAsInt("CLAMAV_MAX_FILE_SIZE_MB", 1024*1024)

	MinioEndpoint = getEnv("MINIO_ENDPOINT", "localhost:9000")
	MinioAccessKey = getEnv("MINIO_ACCESS_KEY", "minioadmin")
	MinioSecretKey = getEnv("MINIO_SECRET_KEY", "minioadmin")
	StagingBucket = getEnv("STAGING_BUCKET", "staging")
	CleanBucket = getEnv("CLEAN_BUCKET", "clean")
	QuarantineBucket = getEnv("QUARANTINE_BUCKET", "quarantine")
	UseSSL = getEnvAsBool("USE_SSL", false)
}

func getEnv(key string, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	valStr := os.Getenv(name)
	if val, err := strconv.Atoi(valStr); err == nil {
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
