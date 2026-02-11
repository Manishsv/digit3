package main

import (

	"gin/config"
	handler "gin/handlers"

	"gin/repository"
	"gin/service"
	"gin/utils"
	"gin/web"
	"log"
	"net/http"


	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupRouter(repo repository.ArtifactRepository, config *config.Config) *gin.Engine {
	r := gin.Default()

	// Add debug logging for Minio configuration (excluding secret)
	log.Printf("Initializing Minio storage with config: endpoint=%s, accessKey=%s, bucket=%s, useSSL=%v, readBucket =%v",
		config.MinioEndpoint,
		config.MinioAccessKey,
		config.MinioWriteBucketName,
		config.MinioUseSSL,
		config.MinioReadBucketName)

	minioStorageSvc, err := service.NewMinioStorageService(config.MinioEndpoint, config.MinioAccessKey, config.MinioSecretKey, config.MinioWriteBucketName, config.MinioReadBucketName, config.MinioUseSSL, repo)

	if err != nil {
		panic("Failed to initialize Minio storage: " + err.Error())
	}

	docSvc := service.NewDocumentCategoryService(repo)

	// Initialize the StorageHandler with the MinioStorageService
	sh := &handler.StorageHandler{
		StorageService:  minioStorageSvc, // Inject MinioStorageService
		DocumentService: docSvc,
		ResponseMaker:   &web.ResponseFactory{},
		Util:            &utils.StorageUtil{},
	}


	route := r.Group(config.ServerContextPath)
	v1 := route.Group("/v1/files")


	// Add health endpoint for Kubernetes liveness probe
	r.GET("/filestore/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// Corresponds to handlers.GetFile - retrieves a file by ID

	v1.GET("/:fileStoreId", sh.GetFile)

	// Corresponds to handlers.GetMetaData - retrieves file metadata by ID
	v1.POST("/metadata", gin.WrapF(http.HandlerFunc(sh.GetMetaData)))

	// Corresponds to handlers.GetUrlListByTag - retrieves files/URLs by tag
	v1.POST("/tag", gin.WrapF(http.HandlerFunc(sh.GetUrlListByTag)))

	// Corresponds to handlers.GetUrls - retrieves file URLs by a list of IDs
	v1.GET("/download-urls", gin.WrapF(http.HandlerFunc(sh.GetUrls)))

	// Corresponds to handlers.StoreFiles - uploads/stores new files
	v1.POST("/upload", gin.WrapF(http.HandlerFunc(sh.StoreFiles)))

	// To Do : think of having rate-limiting or Capacity-limiting for this endpoint
	v1.POST("/upload-url", sh.GetUploadUrl)

	// confirms the file uploaded via upload_url
	v1.POST("/confirm-upload", sh.ConfirmUpload)

	// create a doc category
	v1.POST("/document-categories", sh.CreateDocCategory)

	// get all doc categories
	v1.GET("/document-categories", sh.GetDocCategoryList)

	// get doc category by code
	v1.GET("/document-categories/:docCode", sh.GetDocCategoryByCode)

	// update doc category by code
	v1.PUT("/document-categories/:docCode", sh.UploadDocCategoryByCode)

	// delete doc category by code
	v1.DELETE("/document-categories/:docCode", sh.DeleteDocCategoryByCode)


	return r
}

func main() {

	// if err := godotenv.Load(); err != nil {
	// 	log.Printf("Warning: .env file not found")
	// }

	// Get configurations
	Config := config.NewConfig()

	// Run database migrations before starting the application
	connectionString := Config.GetConnectionString()
	migrationDB, err := sqlx.Connect("postgres", connectionString)
	if err != nil {

		log.Fatalf("Failed to connect to database : %v", err)
	}
	defer migrationDB.Close()

	// Initialize GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(gormpostgres.Open(Config.GetConnectionString()), gormConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	repo := repository.NewPostgresArtifactRepository(db)

	r := setupRouter(repo, Config)
	r.Run()
}
