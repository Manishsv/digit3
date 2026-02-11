package routes

import (
	"template-config/internal/config"
	"template-config/internal/handlers"
	"template-config/internal/repository"
	"template-config/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(db *gorm.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	// Initialize dependencies
	repo := repository.NewTemplateConfigRepository(db)
	svc := service.NewTemplateConfigService(repo)
	handler := handlers.NewTemplateConfigHandler(svc)

	// API routes
	api := router.Group(cfg.ServerContextPath)
	{
		// Template config management routes
		templateConfig := api.Group("/v1/config")
		{
			templateConfig.POST("", handler.CreateTemplateConfig)
			templateConfig.PUT("", handler.UpdateTemplateConfig)
			templateConfig.GET("", handler.SearchTemplateConfigs)
			templateConfig.DELETE("", handler.DeleteTemplateConfig)
		}

		// Template config render route
		api.POST("/v1/render", handler.RenderTemplateConfig)
	}

	return router
}
