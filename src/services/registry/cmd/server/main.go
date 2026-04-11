package main

import (
	"log"

	"registry-service/internal/audit"
	"registry-service/internal/clients"
	"registry-service/internal/config"
	"registry-service/internal/database"
	"registry-service/internal/handlers"
	"registry-service/internal/idgen"
	"registry-service/internal/middleware"
	"registry-service/internal/repository"
	"registry-service/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db := database.Initialize(cfg.DatabaseURL())
	repo := repository.NewRegistryRepository(db)

	auditor := audit.NewNoopAuditor()
	if cfg.VaultRequired && (cfg.Vault.Address == "" || cfg.Vault.Token == "") {
		log.Fatal("vault is required but VAULT_ADDRESS or VAULT_TOKEN not provided")
	}
	if cfg.Vault.Address != "" && cfg.Vault.Token != "" {
		vaultSigner, err := audit.NewVaultSigner(audit.VaultConfig{
			Address:      cfg.Vault.Address,
			Token:        cfg.Vault.Token,
			TransitMount: cfg.Vault.TransitMount,
			KeyPrefix:    cfg.Vault.KeyPrefix,
			Timeout:      cfg.Vault.Timeout,
		})
		if err != nil {
			log.Fatalf("failed to initialize vault signer: %v", err)
		}
		auditor = audit.NewManager(repo, vaultSigner)
	} else if cfg.VaultRequired {
		log.Fatal("vault is required but configuration is incomplete")
	}

	idGenerator := idgen.NewClient(cfg.IDGen)
	externalResolver := clients.NewExternalRegistryClient(cfg.ExternalRegs)
	svc := service.NewRegistryService(repo, auditor, idGenerator, externalResolver)
	handler := handlers.NewRegistryHandler(svc)

	router := gin.Default()
	router.Use(middleware.TenantMiddleware())
	router.Use(middleware.ClientMiddleware())
	router.Use(middleware.CORSMiddleware())

	v1 := router.Group("/registry/v1")
	{
		schemaRoutes := v1.Group("/schema")
		{
			schemaRoutes.POST("", handler.CreateSchema)
			schemaRoutes.GET("", handler.ListSchemas)
			schemaRoutes.GET("/:schemaCode", handler.GetSchema)
			schemaRoutes.PUT("/:schemaCode", handler.UpdateSchema)
			schemaRoutes.DELETE("/:schemaCode", handler.DeleteSchema)
			schemaRoutes.POST("/:schemaCode/_isExist", handler.IsExist)

			dataRoutes := schemaRoutes.Group("/:schemaCode/data")
			{
				dataRoutes.POST("", handler.CreateData)
				dataRoutes.GET("", handler.GetData)
				dataRoutes.PUT("", handler.UpdateData)
				dataRoutes.DELETE("/:id", handler.DeleteData)
				dataRoutes.GET("/_exists", handler.DataExists)
				dataRoutes.GET("/_registry", handler.GetByRegistryID)
				dataRoutes.GET("/_verify", handler.VerifyDataSignature)
				dataRoutes.POST("/_search", handler.SearchData)
			}
		}
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	log.Printf("Starting registry service on port %s", cfg.Port)
	log.Fatal(router.Run(":" + cfg.Port))
}
