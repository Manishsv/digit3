package models

type UploadResponse struct {
	PreSignedURL string `json:"preSignedUrl" binding:"required`
	FileStoreId  string `json:"fileStoreId" binding:"required"`
}
