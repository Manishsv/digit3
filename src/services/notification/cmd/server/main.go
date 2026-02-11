package main

import (
	"fmt"
	"log"
	database "notification/db"
	"notification/internal/config"
	"notification/internal/messaging"
	"notification/internal/routes"
	"notification/internal/service"
	"notification/internal/validators"
	"strings"
)

func buildPostgresDSN(cfg *config.Config) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
	)
}

func startConsumer(cfg *config.Config, emailService *service.EmailService, smsService *service.SMSService) {
	var consumer messaging.Consumer
	switch strings.ToUpper(cfg.MessageBrokerType) {
	case "KAFKA":
		consumer = messaging.NewKafkaConsumer(cfg, emailService, smsService)
	case "REDIS":
		consumer = messaging.NewRedisConsumer(cfg, emailService, smsService)
	default:
		log.Fatalf("unsupported queue type: %s", cfg.MessageBrokerType)
	}

	if err := consumer.Start(); err != nil {
		log.Fatalf("failed to start consumer: %v", err)
	}
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Build DSN
	dsn := buildPostgresDSN(cfg)

	// Setup database
	dbConn, err := database.ConnectDSN(dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Register custom validators before setting up routes
	validators.RegisterCustomValidators()

	// Setup routes
	router, emailService, smsService := routes.SetupRoutes(dbConn, cfg)

	// Setup message broker
	if cfg.MessageBrokerEnabled {
		go startConsumer(cfg, emailService, smsService)
	}

	// Start server
	log.Printf("Starting server on :%s", cfg.HTTPPort)
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
