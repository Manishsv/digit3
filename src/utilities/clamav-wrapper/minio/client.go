package minio

import (
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"clamav-wrapper/config"
)

var Client *minio.Client

func Init() {
	var err error
	Client, err = minio.New(config.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.MinioAccessKey, config.MinioSecretKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		log.Fatalf("MinIO init failed: %v", err)
	}
}
