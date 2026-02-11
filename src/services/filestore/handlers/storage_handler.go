// handler/storage_handler.go
package handler

import (
	"fmt"
	"gin/models"
	"gin/service"
	"gin/utils"
	"gin/web"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// isEmpty checks if a struct is empty (all fields are zero values)
func isEmpty(obj interface{}) bool {
	return reflect.DeepEqual(obj, models.DocumentCategory{})
}

// validateFilesAgainstCategory validates uploaded files against document category rules
func (h *StorageHandler) validateFilesAgainstCategory(files []*multipart.FileHeader, module, tenantId string) error {
	if module == "" {
		log.Printf("No module specified, skipping validation for tenant: %s", tenantId)
		return nil // Skip validation if no module specified
	}

	log.Printf("Validating %d files for module: %s, tenant: %s", len(files), module, tenantId)

	// Get document category for the module
	docCategories, err := h.DocumentService.SearchDocumentCategories("", "", "", tenantId)
	if err != nil {
		return fmt.Errorf("failed to fetch document categories: %w", err)
	}

	log.Printf("Found %d document categories for tenant: %s", len(docCategories), tenantId)

	// Find the category that matches the module
	var category *models.DocumentCategory
	for i := range docCategories {
		if docCategories[i].Type == module {
			category = &docCategories[i]
			log.Printf("Found matching category: %s (code: %s) for module: %s", category.Type, category.Code, module)
			break
		}
	}

	if category == nil {
		log.Printf("No document category found for module: %s, tenant: %s - skipping validation", module, tenantId)
		return nil // No category found for this module, skip validation
	}

	// Validate each file against the category rules
	for _, file := range files {
		if err := h.validateSingleFile(file, category); err != nil {
			return fmt.Errorf("file '%s' validation failed: %w", file.Filename, err)
		}
	}

	return nil
}

// validateSingleFile validates a single file against document category rules
func (h *StorageHandler) validateSingleFile(file *multipart.FileHeader, category *models.DocumentCategory) error {
	log.Printf("Validating file: %s (size: %d bytes) against category: %s", file.Filename, file.Size, category.Code)

	// Check if category is active
	if !category.IsActive {
		return fmt.Errorf("document category '%s' is not active", category.Code)
	}

	// Validate file format
	if len(category.AllowedFormats) > 0 {
		fileExt := strings.ToLower(filepath.Ext(file.Filename))
		if fileExt == "" {
			fileExt = "." + strings.ToLower(file.Header.Get("Content-Type"))
		}

		formatAllowed := false
		for _, allowedFormat := range category.AllowedFormats {
			allowedExt := strings.ToLower(allowedFormat)
			if !strings.HasPrefix(allowedExt, ".") {
				allowedExt = "." + allowedExt
			}
			if fileExt == allowedExt {
				formatAllowed = true
				break
			}
		}

		if !formatAllowed {
			return fmt.Errorf("file format '%s' is not allowed. Allowed formats: %v", fileExt, category.AllowedFormats)
		}
	}

	// Validate file size
	if category.MinSize != "" || category.MaxSize != "" {
		fileSize := file.Size

		if category.MinSize != "" {
			minSize, err := h.parseSize(category.MinSize)
			if err != nil {
				return fmt.Errorf("invalid minSize format in category: %w", err)
			}
			if fileSize < minSize {
				return fmt.Errorf("file size %d bytes is below minimum allowed size %d bytes", fileSize, minSize)
			}
		}

		if category.MaxSize != "" {
			maxSize, err := h.parseSize(category.MaxSize)
			if err != nil {
				return fmt.Errorf("invalid maxSize format in category: %w", err)
			}
			if fileSize > maxSize {
				return fmt.Errorf("file size %d bytes exceeds maximum allowed size %d bytes", fileSize, maxSize)
			}
		}
	}

	return nil
}

// parseSize converts size string (e.g., "1MB", "500KB") to bytes
func (h *StorageHandler) parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)

	// Handle numeric values (assume bytes)
	if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		return size, nil
	}

	// Handle size with units
	var multiplier int64 = 1
	var size int64

	if strings.HasSuffix(sizeStr, "KB") || strings.HasSuffix(sizeStr, "kb") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(strings.TrimSuffix(sizeStr, "KB"), "kb")
	} else if strings.HasSuffix(sizeStr, "MB") || strings.HasSuffix(sizeStr, "mb") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(strings.TrimSuffix(sizeStr, "MB"), "mb")
	} else if strings.HasSuffix(sizeStr, "GB") || strings.HasSuffix(sizeStr, "gb") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(strings.TrimSuffix(sizeStr, "GB"), "gb")
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	return size * multiplier, nil
}

// validateUploadRequest validates upload request against document category rules
func (h *StorageHandler) validateUploadRequest(req models.UploadRequest, tenantId string) error {
	if req.Module == "" {
		log.Printf("No module specified in upload request, skipping validation for tenant: %s", tenantId)
		return nil // Skip validation if no module specified
	}

	log.Printf("Validating upload request for file: %s, module: %s, tenant: %s", req.FileName, req.Module, tenantId)

	// Get document category for the module
	docCategories, err := h.DocumentService.SearchDocumentCategories("", "", "", tenantId)
	if err != nil {
		return fmt.Errorf("failed to fetch document categories: %w", err)
	}

	// Find the category that matches the module
	var category *models.DocumentCategory
	for i := range docCategories {
		if docCategories[i].Type == req.Module {
			category = &docCategories[i]
			break
		}
	}

	if category == nil {
		return nil // No category found for this module, skip validation
	}

	// Check if category is active
	if !category.IsActive {
		return fmt.Errorf("document category '%s' is not active", category.Code)
	}

	// Validate file format
	if len(category.AllowedFormats) > 0 {
		fileExt := strings.ToLower(filepath.Ext(req.FileName))
		if fileExt == "" {
			return fmt.Errorf("file must have a valid extension")
		}

		formatAllowed := false
		for _, allowedFormat := range category.AllowedFormats {
			allowedExt := strings.ToLower(allowedFormat)
			if !strings.HasPrefix(allowedExt, ".") {
				allowedExt = "." + allowedExt
			}
			if fileExt == allowedExt {
				formatAllowed = true
				break
			}
		}

		if !formatAllowed {
			return fmt.Errorf("file format '%s' is not allowed. Allowed formats: %v", fileExt, category.AllowedFormats)
		}
	}

	return nil
}

type StorageHandler struct {
	StorageService  service.StorageService
	DocumentService *service.DocumentCategoryService
	ResponseMaker   *web.ResponseFactory
	Util            *utils.StorageUtil
}

// GET /v1/files/:filestoreId
func (h *StorageHandler) GetFile(c *gin.Context) {
	tenantId := c.Query("tenantId")
	fileStoreId := c.Param("fileStoreId")

	resource, err := h.StorageService.Retrieve(fileStoreId, tenantId)
	if err != nil {
		log.Printf("Error while retrieving file: %v", err)
		c.String(http.StatusInternalServerError, "Error retrieving file: %v", err)
		return
	}

	defer func() {
		if closer, ok := resource.Resource.(io.Closer); ok {
			closer.Close()
		}
	}()

	fileName := resource.FileName[strings.LastIndex(resource.FileName, "/")+1:]

	// Set headers for file download
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", resource.ContentType)
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Accept-Ranges", "bytes")

	if _, err := io.Copy(c.Writer, resource.Resource); err != nil {
		log.Printf("Error while writing file to response: %v", err)

	}
}

// GET /v1/files/metadata
func (h *StorageHandler) GetMetaData(w http.ResponseWriter, r *http.Request) {
	tenantId := r.URL.Query().Get("tenantId")
	fileStoreId := r.URL.Query().Get("fileStoreId")

	resource, err := h.StorageService.Retrieve(fileStoreId, tenantId)
	if err != nil {
		http.Error(w, "Error fetching metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}
	resource.Resource = nil // Remove actual file content
	(&utils.CommonUtil{}).WriteJSON(w, resource, http.StatusOK)
}

// GET /v1/files/tag
func (h *StorageHandler) GetUrlListByTag(w http.ResponseWriter, r *http.Request) {
	tenantId := r.URL.Query().Get("tenantId")
	tag := r.URL.Query().Get("tag")

	fileInfoList := h.StorageService.RetrieveByTag(tag, tenantId)
	resp := h.ResponseMaker.GetFilesByTagResponse(fileInfoList)
	(&utils.CommonUtil{}).WriteJSON(w, resp, http.StatusOK)
}

// POST /v1/files
func (h *StorageHandler) StoreFiles(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}
	tenantId := r.FormValue("tenantId")
	module := r.FormValue("module")
	tag := r.FormValue("tag")
	requestInfo := r.FormValue("requestInfo")

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		http.Error(w, "No files provided for upload", http.StatusBadRequest)
		return
	}

	reqInfo := h.Util.GetRequestInfo(requestInfo)

	// Validate files against document category rules before saving
	if err := h.validateFilesAgainstCategory(files, module, tenantId); err != nil {
		http.Error(w, "File validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	fileStoreIds, err := h.StorageService.Save(files, module, tag, tenantId, reqInfo)
	if err != nil {
		http.Error(w, "Error saving files: "+err.Error(), http.StatusInternalServerError)
		return
	}
	resp := h.getStorageResponse(fileStoreIds, tenantId)
	(&utils.CommonUtil{}).WriteJSON(w, resp, http.StatusCreated)
}

// POST /upload-url
func (h *StorageHandler) GetUploadUrl(c *gin.Context) {
	tenantId := c.Request.Header.Get("X-Tenant-ID")
	var req models.UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate upload request against document category rules
	if err := h.validateUploadRequest(req, tenantId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Upload validation failed: " + err.Error()})
		return
	}

	resp, err := h.StorageService.GetUploadUrl(tenantId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /confirm-upload
func (h *StorageHandler) ConfirmUpload(c *gin.Context) {
	tenantId := c.Request.Header.Get("X-Tenant-ID")
	var req models.ConfirmUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.StorageService.ConfirmUpload(tenantId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /
func (h *StorageHandler) getStorageResponse(fileStoreIds []string, tenantId string) models.StorageResponse {
	files := make([]models.File, 0, len(fileStoreIds))
	for _, id := range fileStoreIds {
		files = append(files, models.File{FileStoreID: id, TenantID: tenantId})
	}
	return models.StorageResponse{Files: files}
}

// GET /url
func (h *StorageHandler) GetUrls(w http.ResponseWriter, r *http.Request) {
	tenantId := r.URL.Query().Get("tenantId")
	fileStoreIdsStr := r.URL.Query().Get("fileStoreIds")

	var fileStoreIds []string
	if fileStoreIdsStr != "" {
		fileStoreIds = strings.Split(fileStoreIdsStr, ",")
	}

	if len(fileStoreIds) == 0 {
		(&utils.CommonUtil{}).WriteJSON(w, map[string]interface{}{}, http.StatusOK)
		return
	}
	maps := h.StorageService.GetUrls(tenantId, fileStoreIds)
	responses := []models.FileStoreResponse{}
	for id, url := range maps {
		responses = append(responses, models.FileStoreResponse{ID: id, URL: url})
	}
	responseMap := map[string]interface{}{
		"fileStoreIds": responses,
	}
	for k, v := range maps {
		responseMap[k] = v
	}
	(&utils.CommonUtil{}).WriteJSON(w, responseMap, http.StatusOK)
}

// POST /document-categories
func (h *StorageHandler) CreateDocCategory(c *gin.Context) {
	tenantId := c.Request.Header.Get("X-Tenant-ID")
	var req models.DocumentCategory
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.DocumentService.CreateDocumentCategory(req, tenantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /document-categories
func (h *StorageHandler) GetDocCategoryList(c *gin.Context) {

	tenantId := c.Request.Header.Get("X-Tenant-ID")
	docType := c.Request.URL.Query().Get("type")
	docCode := c.Request.URL.Query().Get("docCode")
	isSensitive := c.Request.URL.Query().Get("isSensitive")

	resp, err := h.DocumentService.SearchDocumentCategories(docType, docCode, isSensitive, tenantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /document-categories/{docCode}
func (h *StorageHandler) GetDocCategoryByCode(c *gin.Context) {
	tenantId := c.Request.Header.Get("X-Tenant-ID")
	docCode := c.Param("docCode")
	resp, err := h.DocumentService.GetDocumentCategoryByCode(docCode, tenantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if isEmpty(resp) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document category not found"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// PUT /document-categories/{docCode}
func (h *StorageHandler) UploadDocCategoryByCode(c *gin.Context) {
	tenantId := c.Request.Header.Get("X-Tenant-ID")
	docCode := c.Param("docCode")

	// Parse the request body
	var req models.DocumentCategory
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch the existing document category
	existing, err := h.DocumentService.GetDocumentCategoryByCode(docCode, tenantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Document category not found"})
		return
	}

	if isEmpty(existing) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document category not found"})
		return
	}

	// Save the updated document category
	resp, err := h.DocumentService.UpdateDocumentCategory(existing, docCode, tenantId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// DELETE /document-categories/{docCode}
func (h *StorageHandler) DeleteDocCategoryByCode(c *gin.Context) {
	tenantId := c.Request.Header.Get("X-Tenant-ID")
	docCode := c.Param("docCode")
	resp, err := h.DocumentService.DeleteDocumentCategory(docCode, tenantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
