package routes

import (
	"strings"

	"idgen/internal/config"
	"idgen/internal/handlers"
	"idgen/internal/repository"
	"idgen/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(db *gorm.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	// Initialize dependencies
	repo := repository.NewIDGenRepository(db)
	svc := service.NewIDGenService(repo)
	handler := handlers.NewIDGenHandler(svc)

	// Liveness for Docker / load balancers (no auth, no tenant).
	base := strings.TrimSpace(cfg.ServerContextPath)
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = "/idgen"
	}
	router.GET(base+"/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "idgen"})
	})

	// API routes
	api := router.Group(cfg.ServerContextPath)
	{
		// ID Gen Template management routes
		idGenTemplate := api.Group("v1/template")
		{
			idGenTemplate.POST("", handler.CreateTemplate)
			idGenTemplate.PUT("", handler.UpdateTemplate)
			idGenTemplate.GET("", handler.SearchTemplates)
			idGenTemplate.DELETE("", handler.DeleteTemplate)
		}

		api.POST("v1/generate", handler.GenerateID)
	}

	return router
}
