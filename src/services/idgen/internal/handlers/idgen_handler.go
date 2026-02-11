package handlers

import (
	"idgen/internal/models"
	"idgen/internal/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type IDGenHandler struct {
	svc *service.IDGenService
}

func NewIDGenHandler(svc *service.IDGenService) *IDGenHandler {
	return &IDGenHandler{svc: svc}
}

func getTenantIDFromHeader(c *gin.Context) string {
	return c.GetHeader("X-Tenant-ID")
}

func getClientIDFromHeader(c *gin.Context) string {
	return c.GetHeader("X-Client-ID")
}

// RegisterTemplate handles POST /template
func (h *IDGenHandler) CreateTemplate(c *gin.Context) {
	var req models.IDGenTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Params:  []string{err.Error()},
		})
		return
	}

	req.TenantID = getTenantIDFromHeader(c)
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	req.AuditDetails.CreatedBy = getClientIDFromHeader(c)

	idgenTemplateDB, err := h.svc.CreateTemplate(&req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, models.Error{
				Code:        "CONFLICT",
				Message:     "Template already exists",
				Description: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_SERVER_ERROR",
			Message:     "Failed to create template",
			Description: err.Error(),
		})
		return
	}
	dto, err := idgenTemplateDB.ToDTO()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_SERVER_ERROR",
			Message:     "Failed to convert template",
			Description: err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, dto)
}

// UpdateTemplate handles PUT /template
func (h *IDGenHandler) UpdateTemplate(c *gin.Context) {
	var req models.IDGenTemplate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Params:  []string{err.Error()},
		})
		return
	}

	req.TenantID = getTenantIDFromHeader(c)
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	req.AuditDetails.LastModifiedBy = getClientIDFromHeader(c)

	idgenTemplateDB, err := h.svc.UpdateTemplate(&req)
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			c.JSON(http.StatusNotFound, models.Error{
				Code:        "NOT_FOUND",
				Message:     "Template not found",
				Description: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_SERVER_ERROR",
			Message:     "Failed to update template",
			Description: err.Error(),
		})
		return
	}

	dto, err := idgenTemplateDB.ToDTO()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_SERVER_ERROR",
			Message:     "Failed to convert template",
			Description: err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, dto)
}

// SearchTemplates handles GET /template
func (h *IDGenHandler) SearchTemplates(c *gin.Context) {
	var search models.IDGenTemplateSearch
	if err := c.ShouldBindQuery(&search); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid query parameters",
			Description: err.Error(),
		})
		return
	}
	if idsStr := c.Query("ids"); idsStr != "" {
		search.IDs = strings.Split(idsStr, ",")
	}
	search.TenantID = getTenantIDFromHeader(c)
	if search.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	templateDBList, err := h.svc.SearchTemplates(&search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_SERVER_ERROR",
			Message:     "Failed to search templates",
			Description: err.Error(),
		})
		return
	}

	// Map to API models
	templateList := make([]models.IDGenTemplateWithVersion, 0, len(templateDBList))
	for _, templateDB := range templateDBList {
		dto, err := templateDB.ToDTO()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.Error{
				Code:        "INTERNAL_SERVER_ERROR",
				Message:     "Failed to convert template",
				Description: err.Error(),
			})
			return
		}
		templateList = append(templateList, dto)
	}
	c.JSON(http.StatusOK, templateList)
}

// DeleteTemplate handles DELETE /template
func (h *IDGenHandler) DeleteTemplate(c *gin.Context) {
	var deleteReq models.IDGenTemplateDelete
	if err := c.ShouldBindQuery(&deleteReq); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Invalid query parameters",
			Description: err.Error(),
		})
		return
	}
	deleteReq.TenantID = getTenantIDFromHeader(c)
	if deleteReq.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	if err := h.svc.DeleteTemplate(deleteReq.TemplateCode, deleteReq.TenantID, deleteReq.Version); err != nil {
		if strings.Contains(err.Error(), "record not found") {
			c.JSON(http.StatusNotFound, models.Error{
				Code:        "NOT_FOUND",
				Message:     "Template not found",
				Description: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:        "INTERNAL_SERVER_ERROR",
			Message:     "Failed to delete template config",
			Description: err.Error(),
		})
		return
	}
	c.Status(http.StatusOK)
}

func (h *IDGenHandler) GenerateID(c *gin.Context) {
	var req models.GenerateIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:    "BAD_REQUEST",
			Message: "Invalid request body",
			Params:  []string{err.Error()},
		})
		return
	}

	req.TenantID = getTenantIDFromHeader(c)
	if req.TenantID == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing required tenantId",
			Description: "Request must include tenantId in headers",
		})
		return
	}

	response, err := h.svc.GenerateID(req.TenantID, req.TemplateCode, req.Variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Error{
			Code:    "GENERATION_FAILED",
			Message: "Failed to generate ID",
			Params:  []string{err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, response)
}
