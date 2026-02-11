package service

import (
	"fmt"
	"gin/models"
	"log"
	"mime/multipart"
	"strings"
)

// LocalStorageService is an in-memory implementation of StorageService.
// TODO: Replace with a persistent storage solution (e.g., MinIO, S3, local filesystem).
type LocalStorageService struct {
	// For now, we'll store files in memory. This is NOT suitable for production.
	files      map[string]*models.Resource // Key: fileStoreId
	filesByTag map[string][]string         // Key: tag, Value: list of fileStoreIds
}

// NewLocalStorageService creates a new instance of LocalStorageService.
func NewLocalStorageService() StorageService {
	return &LocalStorageService{
		files:      make(map[string]*models.Resource),
		filesByTag: make(map[string][]string),
	}
}

func (s *LocalStorageService) Retrieve(fileStoreId, tenantId string) (*models.Resource, error) {
	// TODO: Implement tenantId validation/filtering if necessary.
	resource, ok := s.files[fileStoreId]
	if !ok {
		return nil, fmt.Errorf("file with id %s not found", fileStoreId)
	}
	// Note: Returning the direct map entry might not be safe if the resource can be modified.
	// Consider returning a copy if necessary.
	return resource, nil
}

func (s *LocalStorageService) RetrieveByTag(tag, tenantId string) []models.FileInfo {
	// TODO: Implement tenantId validation/filtering if necessary.
	fileStoreIds, ok := s.filesByTag[tag]
	if !ok {
		return []models.FileInfo{}
	}

	var fileInfos []models.FileInfo
	for _, id := range fileStoreIds {
		if resource, exists := s.files[id]; exists {
			// Assuming FileLocation can be derived or is stored similarly.
			// This part might need more context on how FileLocation is determined.
			fileInfos = append(fileInfos, models.FileInfo{
				ContentType: resource.ContentType,
				FileLocation: models.FileLocation{ // Placeholder
					FileStoreID: id,
					TenantID:    tenantId,
					Tag:         tag,
					FileName:    resource.FileName,
				},
				TenantID: tenantId,
			})
		}
	}
	return fileInfos
}

func (s *LocalStorageService) Save(files []*multipart.FileHeader, module, tag, tenantId string, reqInfo interface{}) ([]string, error) {
	var savedFileIds []string
	for _, fileHeader := range files {
		// Extremely basic ID generation. Use UUIDs in a real application.
		fileStoreId := fmt.Sprintf("%s-%s-%s", tenantId, module, fileHeader.Filename)
		fileStoreId = strings.ReplaceAll(fileStoreId, " ", "_") // Basic sanitization

		// In a real scenario, you would read the file content and store it.
		// For this in-memory version, we'll just store metadata.
		// The actual file content (io.Reader) would come from fileHeader.Open()
		// and then be stored appropriately (e.g., written to disk, uploaded to cloud).

		file, err := fileHeader.Open()
		if err != nil {
			// Handle or propagate error, potentially skip this file
			fmt.Printf("Error opening file %s: %v\n", fileHeader.Filename, err)
			continue
		}
		// defer file.Close() // Important to close the file

		resource := &models.Resource{
			ContentType: fileHeader.Header.Get("Content-Type"),
			FileName:    fileHeader.Filename,
			TenantID:    tenantId,
			FileSize:    fmt.Sprintf("%d", fileHeader.Size),
			Resource:    file, // Storing the reader itself. In a real system, you'd process this.
		}
		s.files[fileStoreId] = resource

		s.filesByTag[tag] = append(s.filesByTag[tag], fileStoreId)
		savedFileIds = append(savedFileIds, fileStoreId)
	}
	return savedFileIds, nil
}

func (s *LocalStorageService) GetUrls(tenantId string, fileStoreIds []string) map[string]string {
	// TODO: Implement tenantId validation/filtering if necessary.
	urls := make(map[string]string)
	for _, id := range fileStoreIds {
		if _, ok := s.files[id]; ok {
			// For a local/in-memory service, the "URL" might be a path or a data URI.
			// For a real service, this would generate a signed URL or a direct link.
			// This is a placeholder.
			urls[id] = fmt.Sprintf("/filestore/v1/files/id?fileStoreId=%s&tenantId=%s", id, tenantId)
		}
	}
	return urls
}

func (s *LocalStorageService) GetUploadUrl(tenantId string, req models.UploadRequest) (models.UploadResponse, error) {
	log.Printf("GetUploadUrl called for tenantId: %s, fileName: %s", tenantId, req.FileName)
	return models.UploadResponse{
		PreSignedURL: "", // Not applicable for local storage
		FileStoreId:  "", // You can generate or return a placeholder if needed
	}, nil
}

func (s *LocalStorageService) ConfirmUpload(tenantId string, req models.ConfirmUploadRequest) (models.ConfirmUploadResponse, error) {
	// Implement your logic or return a default response
	return models.ConfirmUploadResponse{}, nil
}
