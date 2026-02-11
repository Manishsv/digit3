package service

import (
	"bytes"
	"context"
	"fmt"
	"gin/models"
	"gin/repository"
	"gin/utils"
	"image/png"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioStorageService implements the StorageService interface using MinIO.
type MinioStorageService struct {
	minioClient     *minio.Client
	writeBucketName string
	readBucketName  string
	commonUtil      *utils.CommonUtil
	source          string
	repository      repository.ArtifactRepository
}

// Consider taking MinIO connection details (endpoint, accessKeyID, secretAccessKey, bucketName) as parameters.
func NewMinioStorageService(endpoint, accessKeyID, secretAccessKey, writeBucketName string, readBucketName string, useSSL bool, repo repository.ArtifactRepository) (StorageService, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize minio client: %w", err)
	}

	fmt.Printf("Connecting to MinIO at endpoint: %s\n", endpoint)

	return &MinioStorageService{
		minioClient:     minioClient,
		writeBucketName: writeBucketName,
		readBucketName:  readBucketName,
		commonUtil:      &utils.CommonUtil{},
		source:          "minio", // Set the source identifier
		repository:      repo,
	}, nil
}

func (s *MinioStorageService) Retrieve(fileStoreId, tenantId string) (*models.Resource, error) {

	// check filestoreid
	artifact, err := s.repository.FindByFileStoreIdAndTenantId(fileStoreId, tenantId)
	if err != nil {
		return nil, fmt.Errorf("failed to find artifact in database: %w", err)
	}
	if artifact == nil {
		return nil, fmt.Errorf("artifact not found for fileStoreId: %s and tenantId: %s", fileStoreId, tenantId)
	}

	// Create FileLocation from artifact
	fileLocation := models.FileLocation{
		FileStoreID: artifact.FileStoreID,
		FileName:    artifact.FileName,
		FileSource:  artifact.FileSource,
		TenantID:    artifact.TenantID,
	}

	// Use the new readFile function
	return s.readFile(fileLocation)
}

func (s *MinioStorageService) GetUploadUrl(tenantId string, req models.UploadRequest) (models.UploadResponse, error) {

	// To Do: Move this from config
	expiry := time.Hour * 1

	folderName := s.commonUtil.GetFolderName(req.Module, tenantId)
	fileLocation := s.commonUtil.BuildFileLocation(req.FileName, req.Module, tenantId, req.Tag, folderName)
	artifact := s.commonUtil.BuildArtifact(new(multipart.FileHeader), fileLocation, "")
	artifactEntity := s.commonUtil.MapArtifactsToEntities([]models.Artifact{artifact}, s.source)

	// Save entities to database
	if len(artifactEntity) > 0 {
		_, err := s.repository.SaveAll(artifactEntity)
		if err != nil {
			log.Printf("failed to save artifacts to database: %v", err)
			return models.UploadResponse{}, fmt.Errorf("failed to save artifacts to database: %w", err)
		}
	}

	// Extract the file name with path (remove bucket name)
	index := strings.Index(fileLocation.FileName, "/")
	if index == -1 {
		log.Printf("Invalid file name format: %s", fileLocation.FileName)
	}
	fileNameWithPath := fileLocation.FileName[index+1:]

	// Generate presigned URL for PUT operation
	presignedURL, err := s.minioClient.PresignedPutObject(context.Background(), s.writeBucketName, fileNameWithPath, expiry)
	if err != nil {
		log.Printf("Failed to generate presigned upload URL: %v", err)
		return models.UploadResponse{}, err
	}

	// TODO: Return the URL through a channel or callback
	log.Printf("Generated presigned URL: %s", presignedURL.String())

	return models.UploadResponse{
		PreSignedURL: presignedURL.String(),
		FileStoreId:  artifact.FileLocation.FileStoreID,
	}, nil
}

func (s *MinioStorageService) RetrieveByTag(tag, tenantId string) []models.FileInfo {
	// Query the database for artifacts with the given tag and tenantId
	artifacts, err := s.repository.FindByTagAndTenantId(tag, tenantId)
	if err != nil {
		log.Printf("Error retrieving artifacts by tag: %v", err)
		return []models.FileInfo{}
	}

	// Convert artifacts to FileInfo
	fileInfos := make([]models.FileInfo, 0, len(artifacts))
	for _, artifact := range artifacts {
		fileInfo := models.FileInfo{
			ContentType: artifact.ContentType,
			FileLocation: models.FileLocation{
				FileStoreID: artifact.FileStoreID,
				FileName:    artifact.FileName,
				FileSource:  artifact.FileSource,
				Module:      artifact.Module,
				Tag:         artifact.Tag,
				TenantID:    artifact.TenantID,
			},
			TenantID: artifact.TenantID,
		}
		fileInfos = append(fileInfos, fileInfo)
	}

	return fileInfos
}

func (s *MinioStorageService) Save(files []*multipart.FileHeader, module, tag, tenantId string, reqInfo interface{}) ([]string, error) {
	log.Printf("Save called for tenantId: %s, module: %s, tag: %s", tenantId, module, tag)
	artifacts := s.commonUtil.MapFileToArtifact(files, module, tag, tenantId)

	// Map artifacts to entities using the new function
	artifactEntities := s.commonUtil.MapArtifactsToEntities(artifacts, s.source)

	// Save entities to database
	if len(artifactEntities) > 0 {
		_, err := s.repository.SaveAll(artifactEntities)
		if err != nil {
			return nil, fmt.Errorf("failed to save artifacts to database: %w", err)
		}
	}

	return s.saveArtifacts(artifacts)
}

// saveArtifacts saves a list of artifacts to MinIO storage
func (s *MinioStorageService) saveArtifacts(artifacts []models.Artifact) ([]string, error) {
	var savedFileIds []string

	for _, artifact := range artifacts {
		fileLocation := artifact.FileLocation
		completeName := fileLocation.FileName

		// Extract the file name with path (remove bucket name)
		index := strings.Index(completeName, "/")
		if index == -1 {
			log.Printf("Invalid file name format: %s", completeName)
			continue
		}
		fileNameWithPath := completeName[index+1:]

		// Save main file
		src, err := artifact.MultipartFile.Open()
		if err != nil {
			log.Printf("Error opening file %s: %v", fileNameWithPath, err)
			continue
		}
		defer src.Close()

		contentType := artifact.MultipartFile.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		userMetadata := map[string]string{
			"originalFilename": artifact.MultipartFile.Filename,
			"tag":              fileLocation.Tag,
			"module":           fileLocation.Module,
			"tenantId":         fileLocation.TenantID,
		}

		uploadInfo, err := s.minioClient.PutObject(context.Background(), s.writeBucketName, fileNameWithPath, src, artifact.MultipartFile.Size, minio.PutObjectOptions{
			ContentType:  contentType,
			UserMetadata: userMetadata,
		})
		if err != nil {
			log.Printf("Error uploading file %s to MinIO: %v", fileNameWithPath, err)
			return nil, fmt.Errorf("failed to upload file %s: %w", fileNameWithPath, err)
		}

		log.Printf("Successfully uploaded %s of size %d. ETag: %s, VersionID: %s",
			fileNameWithPath, uploadInfo.Size, uploadInfo.ETag, uploadInfo.VersionID)

		// Save thumbnail images if they exist
		if len(artifact.ThumbnailImages) > 0 {
			if err := s.saveThumbnailImages(artifact); err != nil {
				log.Printf("Error saving thumbnail images for %s: %v", fileNameWithPath, err)
				// Continue with the main file even if thumbnail saving fails
			}
		}

		// Set file source for database persistence
		fileLocation.FileSource = s.source

		savedFileIds = append(savedFileIds, fileLocation.FileStoreID)
	}

	return savedFileIds, nil
}

// saveThumbnailImages saves the thumbnail images for an artifact
func (s *MinioStorageService) saveThumbnailImages(artifact models.Artifact) error {
	basePath := artifact.FileLocation.FileName
	index := strings.Index(basePath, "/")
	if index == -1 {
		return fmt.Errorf("invalid file name format: %s", basePath)
	}
	basePath = basePath[index+1:]
	basePath = strings.TrimSuffix(basePath, filepath.Ext(basePath))

	for size, img := range artifact.ThumbnailImages {
		// Convert image to bytes
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return fmt.Errorf("failed to encode thumbnail image: %w", err)
		}

		// Create thumbnail path
		thumbnailPath := fmt.Sprintf("%s_%s.png", basePath, size)

		// Upload thumbnail
		_, err := s.minioClient.PutObject(context.Background(), s.writeBucketName, thumbnailPath, &buf, int64(buf.Len()), minio.PutObjectOptions{
			ContentType: "image/png",
			UserMetadata: map[string]string{
				"originalFile":  artifact.FileLocation.FileName,
				"thumbnailSize": size,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to upload thumbnail %s: %w", thumbnailPath, err)
		}
	}

	return nil
}

// isFileAnImage checks if the given file is an image based on its extension
func (s *MinioStorageService) isFileAnImage(fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	imageExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".webp": true,
	}
	return imageExts[ext]
}

// setThumbnailSignedURL generates a presigned URL for the thumbnail version of an image
func (s *MinioStorageService) setThumbnailSignedURL(fileName string) (string, error) {
	// Extract base path without extension
	basePath := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	thumbnailPath := fmt.Sprintf("%s_thumbnail.png", basePath)

	// Generate presigned URL for thumbnail
	expiry := time.Hour * 1
	presignedURL, err := s.minioClient.PresignedGetObject(context.Background(), s.writeBucketName, thumbnailPath, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for thumbnail: %w", err)
	}

	return presignedURL.String(), nil
}

func (s *MinioStorageService) GetUrls(tenantId string, fileStoreIds []string) map[string]string {
	log.Printf("GetUrls called for tenantId: %s, fileStoreIds: %v", tenantId, fileStoreIds)
	urls := make(map[string]string)

	// URLs expire in 1 hour
	expiry := time.Hour * 1

	for _, fileStoreId := range fileStoreIds {
		// First retrieve the artifact to get the file information
		artifact, err := s.repository.FindByFileStoreIdAndTenantId(fileStoreId, tenantId)
		if err != nil {
			log.Printf("Error finding artifact for fileStoreId %s: %v", fileStoreId, err)
			continue
		}

		if artifact == nil {
			log.Printf("No artifact found for fileStoreId %s", fileStoreId)
			continue
		}

		// Extract the file name with path (remove bucket name)
		fileName := artifact.FileName
		if idx := strings.Index(fileName, "/"); idx != -1 {
			fileName = fileName[idx+1:]
		}

		// Generate presigned URL for the main file
		presignedURL, err := s.minioClient.PresignedGetObject(context.Background(), s.readBucketName, fileName, expiry, nil)
		if err != nil {
			log.Printf("Error generating presigned URL for %s: %v", fileName, err)
			continue
		}

		// If it's an image, try to get the thumbnail URL
		if s.isFileAnImage(artifact.FileName) {
			thumbnailURL, err := s.setThumbnailSignedURL(fileName)
			if err != nil {
				log.Printf("Error generating thumbnail URL for %s: %v", fileName, err)
				// Continue with the main file URL even if thumbnail fails
			} else {
				urls[fileStoreId] = thumbnailURL
				continue
			}
		}

		urls[fileStoreId] = presignedURL.String()
	}

	return urls
}

// readFile reads a file from MinIO and returns it as a Resource
func (s *MinioStorageService) readFile(fileLocation models.FileLocation) (*models.Resource, error) {
	if fileLocation.FileSource == "" || fileLocation.FileSource == s.source {
		// Extract filename after the first '/'
		fileName := fileLocation.FileName
		if idx := strings.Index(fileName, "/"); idx != -1 {
			fileName = fileName[idx+1:]
		}

		// Get the object directly from MinIO
		object, err := s.minioClient.GetObject(context.Background(), s.readBucketName, fileName, minio.GetObjectOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get object from MinIO: %w", err)
		}

		// Get object stats
		stat, err := object.Stat()
		if err != nil {
			object.Close()
			return nil, fmt.Errorf("failed to get object stats: %w", err)
		}

		return &models.Resource{
			ContentType: stat.ContentType,
			FileName:    fileLocation.FileName,
			Resource:    object,
			TenantID:    fileLocation.TenantID,
			FileSize:    fmt.Sprintf("%d", stat.Size),
		}, nil
	}

	return nil, fmt.Errorf("unsupported file source: %s", fileLocation.FileSource)
}

func (s *MinioStorageService) ConfirmUpload(tenantId string, req models.ConfirmUploadRequest) (models.ConfirmUploadResponse, error) {

	resource, error := s.Retrieve(req.FileStoreID, tenantId)
	if error != nil {
		// remove the entry from db
		s.repository.DeleteByFileStoreIdAndTenantId(req.FileStoreID, tenantId)
		return models.ConfirmUploadResponse{Status: "INVALID"}, error
	}
	// update the content type in db
	s.repository.UpdateContentType(req.FileStoreID, tenantId, resource.ContentType)
	return models.ConfirmUploadResponse{Status: "VALID"}, error
}
