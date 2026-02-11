package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"boundary/internal/config"
	"boundary/internal/handlers"
	"boundary/internal/repository"
	"boundary/internal/service"
	"boundary/pkg/cache"
	"boundary/pkg/postgres"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// initTracer initializes OpenTelemetry tracing with Jaeger exporter
func initTracer(cfg *config.Config) func() {
	if !cfg.OpenTelemetry.Enabled {
		log.Println("OpenTelemetry tracing is disabled")
		return func() {}
	}

	// Use OTLP HTTP exporter
	exp, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(cfg.OpenTelemetry.OTLPEndpoint), // <-- new field
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Printf("Failed to create OTLP exporter: %v", err)
		return func() {}
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.OpenTelemetry.ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		log.Printf("Failed to create resource: %v", err)
		return func() {}
	}

	// Create trace provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(cfg.OpenTelemetry.SamplingRatio)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	log.Printf("OpenTelemetry tracing initialized with service name: %s", cfg.OpenTelemetry.ServiceName)

	// Return cleanup function
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}
}

func main() {
	// Load configuration from environment variables
	cfg := config.LoadConfig()

	// Initialize OpenTelemetry tracing
	cleanup := initTracer(cfg)
	defer cleanup()

	// Initialize database connection
	db, err := postgres.NewConnection(&cfg.Database)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// Initialize cache
	var cacheImpl cache.Cache
	switch cfg.Cache.Type {
	case "redis":
		cacheImpl = cache.NewRedisCache(cfg.Cache.Redis.Addr, cfg.Cache.Redis.Password, cfg.Cache.Redis.DB)
		log.Println("Using Redis cache")
	default:
		cacheImpl = cache.NewInMemoryCache()
		log.Println("Using in-memory cache")
	}

	// Initialize repositories
	boundaryRepo := repository.NewBoundaryRepository(db, cfg)
	hierarchyRepo := repository.NewBoundaryHierarchyRepository(db, cfg)
	relationshipRepo := repository.NewBoundaryRelationshipRepository(db, cfg)

	// Initialize services
	boundaryService := service.NewBoundaryService(boundaryRepo, cacheImpl)
	hierarchyService := service.NewBoundaryHierarchyService(hierarchyRepo, cacheImpl)
	relationshipService := service.NewBoundaryRelationshipService(relationshipRepo, cacheImpl)

	// Initialize handlers
	boundaryHandler := handlers.NewBoundaryHandler(boundaryService, hierarchyService, relationshipService)

	// Setup Gin router
	router := gin.Default()

	// Add OpenTelemetry middleware for HTTP tracing
	if cfg.OpenTelemetry.Enabled {
		router.Use(otelgin.Middleware(cfg.OpenTelemetry.ServiceName))
	}

	// Register routes at root (no context path prefix)
	api := router.Group(cfg.Server.ContextPath)
	boundaryHandler.RegisterRoutes(api)

	// Start server
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
}
