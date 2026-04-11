package handlers

import (
	"errors"
	"net/http"
	"registry-service/internal/models"
	"registry-service/internal/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RegistryHandler struct {
	service service.RegistryService
}

func NewRegistryHandler(service service.RegistryService) *RegistryHandler {
	return &RegistryHandler{service: service}
}

// Schema handlers
func (h *RegistryHandler) CreateSchema(c *gin.Context) {
	var request models.SchemaRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Get tenantId and clientId from context (set by middleware)
	tenantID := c.GetString("tenantId")
	clientID := c.GetString("clientId")

	// Override request tenantId with header value
	request.TenantID = tenantID

	schema, err := h.service.CreateSchema(&request, clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.Response{
		Success: true,
		Data:    schema,
		Message: "Schema created successfully",
	})
}

func (h *RegistryHandler) GetSchema(c *gin.Context) {
	schemaCode := c.Param("schemaCode")
	tenantID := c.GetString("tenantId")

	var (
		schema *models.Schema
		err    error
	)

	if versionStr := c.Query("version"); versionStr != "" {
		version, convErr := strconv.Atoi(versionStr)
		if convErr != nil {
			c.JSON(http.StatusBadRequest, models.Response{
				Success: false,
				Error:   "Invalid version parameter",
			})
			return
		}
		schema, err = h.service.GetSchemaVersion(tenantID, schemaCode, version)
	} else {
		schema, err = h.service.GetSchema(tenantID, schemaCode)
	}
	if err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Success: false,
			Error:   "Schema not found",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Data:    schema,
	})
}

func (h *RegistryHandler) UpdateSchema(c *gin.Context) {
	schemaCode := c.Param("schemaCode")
	tenantID := c.GetString("tenantId")
	clientID := c.GetString("clientId")

	var request models.SchemaRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Override request tenantId with header value
	request.TenantID = tenantID

	schema, err := h.service.UpdateSchema(tenantID, schemaCode, &request, clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Data:    schema,
		Message: "Schema updated successfully",
	})
}

func (h *RegistryHandler) DeleteSchema(c *gin.Context) {
	schemaCode := c.Param("schemaCode")
	tenantID := c.GetString("tenantId")
	clientID := c.GetString("clientId")

	err := h.service.DeleteSchema(tenantID, schemaCode, clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Success: false,
			Error:   "Schema not found",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: "Schema deleted successfully",
	})
}

func (h *RegistryHandler) ListSchemas(c *gin.Context) {
	tenantID := c.GetString("tenantId")

	schemas, err := h.service.ListSchemas(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Success: false,
			Error:   "Failed to retrieve schemas",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Data:    schemas,
	})
}

// Data handlers
func schemaCodeFromPath(c *gin.Context) (string, bool) {
	code := strings.TrimSpace(c.Param("schemaCode"))
	if code == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "schemaCode path parameter is required",
		})
		return "", false
	}
	return code, true
}

func (h *RegistryHandler) CreateData(c *gin.Context) {
	schemaCode, ok := schemaCodeFromPath(c)
	if !ok {
		return
	}
	tenantID := c.GetString("tenantId")
	clientID := c.GetString("clientId")

	var request models.DataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Override request tenantId with header value
	request.TenantID = tenantID
	request.SchemaCode = schemaCode

	data, err := h.service.CreateData(schemaCode, &request, clientID, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.Response{
		Success: true,
		Data:    data,
		Message: "Data created successfully",
	})
}

func (h *RegistryHandler) GetData(c *gin.Context) {
	schemaCode, ok := schemaCodeFromPath(c)
	if !ok {
		return
	}
	tenantID := c.GetString("tenantId")
	id := c.Query("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "ID query parameter is required",
		})
		return
	}

	data, err := h.service.GetData(tenantID, schemaCode, id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Success: false,
			Error:   "Data not found",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Data:    data,
	})
}

func (h *RegistryHandler) DataExists(c *gin.Context) {
	schemaCode, ok := schemaCodeFromPath(c)
	if !ok {
		return
	}
	tenantID := c.GetString("tenantId")
	id := c.Query("id")

	if strings.TrimSpace(id) == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "ID query parameter is required",
		})
		return
	}

	exists, err := h.service.DataExists(tenantID, schemaCode, id)
	if err != nil {
		httpStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "must be provided") {
			httpStatus = http.StatusBadRequest
		}
		c.JSON(httpStatus, models.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Data: map[string]bool{
			"exists": exists,
		},
	})
}

func (h *RegistryHandler) IsExist(c *gin.Context) {
	schemaCode, ok := schemaCodeFromPath(c)
	if !ok {
		return
	}
	tenantID := c.GetString("tenantId")

	var request models.IsExistRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{Success: false, Error: "Invalid request body: " + err.Error()})
		return
	}
	if strings.TrimSpace(request.TenantID) != "" {
		tenantID = strings.TrimSpace(request.TenantID)
	}

	field := strings.TrimSpace(request.Field)
	if field == "" {
		field = "registryId"
	}
	value := strings.TrimSpace(request.Value)
	if value == "" {
		c.JSON(http.StatusBadRequest, models.Response{Success: false, Error: "value is required"})
		return
	}

	exists, err := h.service.FieldExists(tenantID, schemaCode, field, value)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "required") {
			status = http.StatusBadRequest
		}
		c.JSON(status, models.Response{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Data:    map[string]bool{"exists": exists},
	})
}

func (h *RegistryHandler) UpdateData(c *gin.Context) {
	schemaCode, ok := schemaCodeFromPath(c)
	if !ok {
		return
	}
	id := strings.TrimSpace(c.Query("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "id query parameter is required",
		})
		return
	}
	tenantID := c.GetString("tenantId")
	clientID := c.GetString("clientId")

	var request models.DataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Override request tenantId with header value
	request.TenantID = tenantID
	request.SchemaCode = schemaCode

	if request.Version <= 0 {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "version is required for update",
		})
		return
	}

	data, err := h.service.UpdateData(tenantID, schemaCode, id, &request, clientID, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Data:    data,
		Message: "Data updated successfully",
	})
}

func (h *RegistryHandler) DeleteData(c *gin.Context) {
	schemaCode, ok := schemaCodeFromPath(c)
	if !ok {
		return
	}
	id := c.Param("id")
	tenantID := c.GetString("tenantId")
	clientID := c.GetString("clientId")

	err := h.service.DeleteData(tenantID, schemaCode, id, clientID, nil)
	if err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Success: false,
			Error:   "Data not found",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: "Data deleted successfully",
	})
}

func (h *RegistryHandler) SearchData(c *gin.Context) {
	schemaCode, ok := schemaCodeFromPath(c)
	if !ok {
		return
	}
	tenantID := c.GetString("tenantId")

	var request models.SearchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "Invalid request body: " + err.Error(),
		})
		return
	}

	// Override request values with path params and headers
	request.TenantID = tenantID
	request.SchemaCode = schemaCode

	// Set default pagination if not provided
	if request.Limit == 0 {
		if limitStr := c.Query("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil {
				request.Limit = limit
			}
		} else {
			request.Limit = 100 // Default limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			request.Offset = offset
		}
	}

	data, err := h.service.SearchData(&request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Success: false,
			Error:   "Failed to search data",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Data:    data,
	})
}

func (h *RegistryHandler) VerifyDataSignature(c *gin.Context) {
	schemaCode, ok := schemaCodeFromPath(c)
	if !ok {
		return
	}
	tenantID := c.GetString("tenantId")
	identifier := strings.TrimSpace(c.Query("id"))
	if identifier == "" {
		identifier = strings.TrimSpace(c.Query("registryId"))
	}
	if identifier == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "id or registryId query parameter is required",
		})
		return
	}

	valid, err := h.service.VerifyDataSignature(tenantID, schemaCode, identifier)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, service.ErrAuditLogNotFound):
			status = http.StatusNotFound
		case errors.Is(err, service.ErrSignatureVerificationUnavailable):
			status = http.StatusNotImplemented
		case strings.Contains(err.Error(), "required"):
			status = http.StatusBadRequest
		}
		c.JSON(status, models.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Data: map[string]bool{
			"valid": valid,
		},
	})
}
func (h *RegistryHandler) GetByRegistryID(c *gin.Context) {
	schemaCode, ok := schemaCodeFromPath(c)
	if !ok {
		return
	}
	tenantID := c.GetString("tenantId")
	registryID := strings.TrimSpace(c.Query("registryId"))
	if registryID == "" {
		c.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Error:   "registryId query parameter is required",
		})
		return
	}
	history := strings.EqualFold(strings.TrimSpace(c.Query("history")), "true")

	records, err := h.service.GetRegistryData(tenantID, schemaCode, registryID, history)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		c.JSON(status, models.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	var result interface{}
	if history {
		result = records
	} else if len(records) > 0 {
		result = records[0]
	}

	c.JSON(http.StatusOK, models.Response{
		Success: true,
		Data:    result,
	})
}
