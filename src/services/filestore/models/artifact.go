package models

import (
	"image"
	"mime/multipart"
	"time"
)

// Artifact represents a file artifact with metadata and thumbnails
type Artifact struct {
	FileContentInString string                 // Base64 or raw string content
	MultipartFile       multipart.FileHeader   // Uploaded file reference
	FileLocation        FileLocation           // Storage location details
	ThumbnailImages     map[string]image.Image // Thumbnails by size/type
	CreatedBy           string                 // User ID of creator
	LastModifiedBy      string                 // User ID of last modifier
	CreatedTime         time.Time              // Creation timestamp
	LastModifiedTime    time.Time              // Last modification timestamp
}
