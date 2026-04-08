package handlers

import (
	"fmt"
	"idgen/internal/models"
	"idgen/internal/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Matches V20260405120000__extend_idgen_varchar_columns.sql (was VARCHAR(64)).
const idgenStringMax = 255
const idgenClientIDMaxLegacy = 64 // audit columns were VARCHAR(64); truncate so inserts work before migrate (same as boundary).

func trimTenantID(c *gin.Context) (string, error) {
	t := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
	if t == "" {
		return "", fmt.Errorf("missing X-Tenant-ID")
	}
	if len(t) > idgenStringMax {
		return "", fmt.Errorf("X-Tenant-ID length %d exceeds %d (run idgen DB migration V20260405120000 or shorten realm)", len(t), idgenStringMax)
	}
	return t, nil
}

// Prefer X-Client-ID / X-Client-Id; truncate to idgenClientIDMaxLegacy (safe on unmigrated VARCHAR(64) DB).
func effectiveClientID(c *gin.Context) string {
	cid := strings.TrimSpace(c.GetHeader("X-Client-ID"))
	if cid == "" {
		cid = strings.TrimSpace(c.GetHeader("X-Client-Id"))
	}
	if len(cid) > idgenClientIDMaxLegacy {
		return cid[:idgenClientIDMaxLegacy]
	}
	return cid
}

type IDGenHandler struct {
	svc *service.IDGenService
}

func NewIDGenHandler(svc *service.IDGenService) *IDGenHandler {
	return &IDGenHandler{svc: svc}
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

	tenantID, err := trimTenantID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing or invalid tenant",
			Description: err.Error(),
		})
		return
	}
	req.TenantID = tenantID

	if tc := strings.TrimSpace(req.TemplateCode); tc == "" {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "templateCode is required",
			Description: "templateCode must be non-empty",
		})
		return
	} else if len(tc) > idgenStringMax {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "templateCode too long",
			Description: fmt.Sprintf("templateCode length %d exceeds %d", len(tc), idgenStringMax),
		})
		return
	}
	req.TemplateCode = strings.TrimSpace(req.TemplateCode)

	req.AuditDetails.CreatedBy = effectiveClientID(c)

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

	tenantID, errT := trimTenantID(c)
	if errT != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing or invalid tenant",
			Description: errT.Error(),
		})
		return
	}
	req.TenantID = tenantID

	req.AuditDetails.LastModifiedBy = effectiveClientID(c)

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
	tid, errT := trimTenantID(c)
	if errT != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing or invalid tenant",
			Description: errT.Error(),
		})
		return
	}
	search.TenantID = tid

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
	tid, errT := trimTenantID(c)
	if errT != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing or invalid tenant",
			Description: errT.Error(),
		})
		return
	}
	deleteReq.TenantID = tid

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

	tid, errT := trimTenantID(c)
	if errT != nil {
		c.JSON(http.StatusBadRequest, models.Error{
			Code:        "BAD_REQUEST",
			Message:     "Missing or invalid tenant",
			Description: errT.Error(),
		})
		return
	}
	req.TenantID = tid

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
