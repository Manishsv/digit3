// web/response_factory.go
package web

import (
	"fmt"
	"gin/models"
)

type ResponseFactory struct {
	contextPath string
}

type FileRecord struct {
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
}

type GetFilesByTagResponse struct {
	Files []FileRecord `json:"files"`
}

func NewResponseFactory(contextPath string) *ResponseFactory {
	return &ResponseFactory{
		contextPath: contextPath,
	}
}

func (rf *ResponseFactory) GetFilesByTagResponse(fileInfos []models.FileInfo) GetFilesByTagResponse {
	const format = "%s/v1/files/id?fileStoreId=%s&tenantId=%s"

	fileRecords := make([]FileRecord, len(fileInfos))
	for i, fileInfo := range fileInfos {
		url := fmt.Sprintf(format, rf.contextPath, fileInfo.FileLocation.FileStoreID, fileInfo.TenantID)
		fileRecords[i] = FileRecord{
			URL:         url,
			ContentType: fileInfo.ContentType,
		}
	}

	return GetFilesByTagResponse{Files: fileRecords}
}
