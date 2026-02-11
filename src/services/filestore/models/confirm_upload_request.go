package models

type ConfirmUploadRequest struct {
	FileStoreID string `json:"fileStoreId" binding:"required"`
}
