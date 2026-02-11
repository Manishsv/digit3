package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	localizationv1 "localization/api/proto/localization/v1"
	"localization/configs"
	"localization/internal/cache"
	"localization/internal/core/ports"
	"localization/internal/core/services"
	"localization/internal/handlers"
	dbpostgres "localization/internal/repositories/postgres"
)

func main() {
	// Load application configurations
	config := configs.LoadConfig()

	// Setup GORM database connection
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName, config.DBSSLMode)

	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// Enable connection pooling for better performance
		ConnPool: nil, // Use default connection pool
	})
	if err != nil {
		log.Fatalf("could not connect to the database: %v", err)
	}

	// Get underlying sql.DB for migration runner (migrations still use sqlx)
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("could not get underlying sql.DB: %v", err)
	}

	// Configure connection pool for optimal performance
	sqlDB.SetMaxOpenConns(25)                 // Maximum number of open connections
	sqlDB.SetMaxIdleConns(10)                 // Maximum number of idle connections
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // Maximum connection lifetime

	// Initialize Cache
	var messageCache ports.MessageCache
	cacheType := os.Getenv("CACHE_TYPE")

	if cacheType == "in-memory" {
		messageCache = cache.NewInMemoryMessageCache()
		log.Println("Initialized in-memory cache.")
	} else {
		// Default to Redis
		redisClient := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
			Password: config.RedisPassword,
			DB:       config.RedisDB,
		})
		messageCache = cache.NewRedisMessageCache(redisClient)
		log.Println("Initialized Redis cache.")
	}

	// Initialize repository and service with GORM
	messageRepo := dbpostgres.NewMessageRepository(gormDB)
	messageService := services.NewMessageService(messageRepo, messageCache)

	// Load all messages into memory for the missing messages API
	if err := messageService.LoadAllMessages(context.Background()); err != nil {
		log.Fatalf("Failed to load messages into memory map: %v", err)
	}

	// Setup HTTP server
	httpRouter := gin.Default()
	messageHandler := handlers.NewMessageHandler(messageService)
	apiGroup := httpRouter.Group(config.ServerContextPath)
	messageHandler.RegisterRoutes(apiGroup)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.RESTPort),
		Handler: httpRouter,
	}

	// Setup gRPC server
	grpcServer := grpc.NewServer()
	localizationGRPCServer := handlers.NewGRPCServer(messageService)
	localizationv1.RegisterLocalizationServiceServer(grpcServer, localizationGRPCServer)

	// Start servers
	go func() {
		log.Printf("HTTP server listening on :%d", config.RESTPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.GRPCPort))
		if err != nil {
			log.Fatalf("gRPC server failed to listen: %v", err)
		}
		log.Printf("gRPC server listening on :%d", config.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server shutdown failed: %v", err)
	}
	grpcServer.GracefulStop()
	log.Println("Servers gracefully stopped")
}
