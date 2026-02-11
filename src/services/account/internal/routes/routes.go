package routes

import (
	"account/internal/config"
	"account/internal/handlers"
	"account/internal/keycloak"
	"account/internal/notification"
	"account/internal/repository"
	"account/internal/service"

	"database/sql"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB, cfg *config.Config) {
	// Create Keycloak client
	keycloakClient := keycloak.NewKeycloakClient(cfg.Keycloak.BaseURL, cfg.Keycloak.AdminUser, cfg.Keycloak.AdminPass, &cfg.Keycloak)

	// Create notification client
	notificationClient := notification.NewClient(cfg.Notification.BaseURL)

	// Create router group without context path
	api := r.Group(cfg.Server.ContextPath)

	// Account (4 APIs)
	tenantRepo := repository.NewTenantRepository(db)
	tenantService := service.NewTenantService(tenantRepo, keycloakClient, notificationClient)
	tenantHandler := handlers.NewTenantHandler(tenantService)
	api.POST("/v1", tenantHandler.CreateTenant)    // Create
	api.GET("/v1", tenantHandler.ListTenants)      // Search
	api.PUT("/v1/:id", tenantHandler.UpdateTenant) // Update
	api.DELETE("/v1", tenantHandler.DeleteAccount) // Delete Account (realm + tenant + config)

	// AccountConfig (3 APIs)
	tenantConfigRepo := repository.NewTenantConfigRepository(db)
	documentRepo := repository.NewDocumentRepository(db)
	tenantConfigService := service.NewTenantConfigService(tenantConfigRepo, tenantService, documentRepo)
	tenantConfigHandler := handlers.NewTenantConfigHandler(tenantConfigService, tenantService)
	api.POST("/v1/config", tenantConfigHandler.CreateTenantConfig)    // Create
	api.GET("/v1/config", tenantConfigHandler.ListTenantConfigs)      // Search
	api.PUT("/v1/config/:id", tenantConfigHandler.UpdateTenantConfig) // Update
}
