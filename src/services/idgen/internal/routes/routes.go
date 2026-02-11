package routes

import (
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
