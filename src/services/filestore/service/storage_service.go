package service

import (
	"gin/models"
	"mime/multipart"
)

type StorageService interface {
	Retrieve(fileStoreId, tenantId string) (*models.Resource, error)
	RetrieveByTag(tag, tenantId string) []models.FileInfo
	Save(files []*multipart.FileHeader, module, tag, tenantId string, reqInfo interface{}) ([]string, error)
	GetUrls(tenantId string, fileStoreIds []string) map[string]string
	GetUploadUrl(tenantId string, req models.UploadRequest) (models.UploadResponse, error)
	ConfirmUpload(tenantId string, req models.ConfirmUploadRequest) (models.ConfirmUploadResponse, error)
}
