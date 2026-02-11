package models

type UploadRequest struct {
	FileName    string `json:"fileName" binding:"required"`
	ContentType string `json:"contentType" binding:"required"`
	Module      string `json:"module" binding:"required"`
	Tag         string `json:"tag" binding:"required"`
}
