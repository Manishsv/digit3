package routes

import (
	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/handlers"
	"url-shortener/internal/repository"
	"url-shortener/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(db *gorm.DB, cfg *config.Config, cache cache.Cache) *gin.Engine {
	router := gin.Default()

	// Initialize dependencies
	repo := repository.NewURLShortenerRepository(db, cfg, cache)
	svc := service.NewURLShortenerService(repo, cfg)
	handler := handlers.NewURLShortenerHandler(svc)

	// API routes
	api := router.Group(cfg.ServerContextPath)
	{
		api.POST("/v1/shortener", handler.ShortenURLHandler)
		api.GET("/:key", handler.RedirectURLHandler)
	}

	return router
}
