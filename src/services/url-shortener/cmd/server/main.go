package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"url-shortener/db"
	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/routes"
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

func initCache(cfg *config.Config) cache.Cache {
	if !cfg.CacheEnabled {
		log.Printf("Caching is disabled")
		return cache.NewNoOpCache()
	}

	switch strings.ToUpper(cfg.CacheType) {
	case "REDIS":
		log.Printf("Using Redis cache")
		return cache.NewRedisCache(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.CacheTTL)
	case "INMEMORY":
		log.Printf("Using in-memory cache")
		return cache.NewInMemoryCache()
	default:
		log.Fatalf("unsupported cache type: %s", cfg.CacheType)
	}
	return cache.NewNoOpCache()
}

func validateConfig(cfg *config.Config) error {
	if cfg.ShortKeyMinLength <= 0 {
		return fmt.Errorf("SHORT_KEY_MIN_LENGTH must be > 0")
	}
	if cfg.MaxShortKeyRetries <= 0 {
		return fmt.Errorf("MAX_SHORT_KEY_RETRIES must be > 0")
	}
	if !IsValidURL(cfg.HostName) {
		return fmt.Errorf("HOST_NAME is invalid")
	}
	return nil
}

// IsValidURL returns true if URL is a valid full URL (http or https)
func IsValidURL(u string) bool {
	parsed, err := url.ParseRequestURI(u)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	return true
}

func main() {
	// Load configuration
	cfg := config.Load()

	// after loading cfg
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// Build DSN
	dsn := buildPostgresDSN(cfg)

	// Setup database
	dbConn, err := db.ConnectDSN(dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize cache
	cacheClient := initCache(cfg)

	// Setup routes
	router := routes.SetupRoutes(dbConn, cfg, cacheClient)

	// Start server
	log.Printf("Starting server on :%s", cfg.HTTPPort)
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
