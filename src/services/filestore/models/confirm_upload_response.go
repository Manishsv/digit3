package models

type ConfirmUploadResponse struct {
	Status string `json:"status" binding:"required"`
}
