package handlers

import (
	"account/internal/models"
	"account/internal/service"
	"account/internal/validator"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TenantConfigHandler struct {
	Service       *service.TenantConfigService
	TenantService *service.TenantService
}

func NewTenantConfigHandler(svc *service.TenantConfigService, tenantSvc *service.TenantService) *TenantConfigHandler {
	return &TenantConfigHandler{Service: svc, TenantService: tenantSvc}
}

func (h *TenantConfigHandler) CreateTenantConfig(c *gin.Context) {
	// 1. Parse request body
	var req models.TenantConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "BAD_REQUEST",
				Message:     "Invalid request body",
				Description: err.Error(),
			}},
		})
		return
	}

	// 2. Call service layer for business logic
	err := h.Service.CreateTenantConfigWithValidation(context.Background(), &req.TenantConfig, c.GetHeader("X-Client-Id"))
	if err != nil {
		// Handle specific error types
		if err.Error() == "TENANT_NOT_FOUND" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Errors: []models.Error{{
					Code:    "TENANT_NOT_FOUND",
					Message: "Tenant doesn't exist for given code",
				}},
			})
			return
		}

		if validationErr, ok := err.(*validator.ValidationError); ok {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Errors: []models.Error{{
					Code:    validationErr.Code,
					Message: err.Error(),
				}},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "INTERNAL_ERROR",
				Message:     "Failed to create tenant config",
				Description: err.Error(),
			}},
		})
		return
	}

	// 3. Return success response
	resp := models.TenantConfigResponse{
		TenantConfigs: []models.TenantConfig{req.TenantConfig},
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *TenantConfigHandler) GetTenantConfigByID(c *gin.Context) {
	id := c.Param("id")
	t, err := h.Service.GetTenantConfigByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "INTERNAL_ERROR",
				Message:     "Failed to get tenant config",
				Description: err.Error(),
			}},
		})
		return
	}
	if t == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "NOT_FOUND",
				Message:     "TenantConfig not found",
				Description: "No tenant config found with the given ID",
			}},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"TenantConfig": t,
	})
}

func (h *TenantConfigHandler) ListTenantConfigs(c *gin.Context) {
	// Get query parameters for filtering
	code := c.Query("code")
	name := c.Query("name")

	configs, err := h.Service.ListTenantConfigs(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "INTERNAL_ERROR",
				Message:     "Failed to list tenant configs",
				Description: err.Error(),
			}},
		})
		return
	}
	var filtered []models.TenantConfig
	for _, t := range configs {
		// Apply filters
		if code != "" && t.Code != code {
			continue
		}
		if name != "" && t.Name != name {
			continue
		}

		filtered = append(filtered, *t)
	}
	resp := models.TenantConfigResponse{
		TenantConfigs: filtered,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *TenantConfigHandler) UpdateTenantConfig(c *gin.Context) {
	// 1. Extract ID from path parameter
	id := c.Param("id")

	// 2. Parse request body
	var req models.TenantConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "BAD_REQUEST",
				Message:     "Invalid request body",
				Description: err.Error(),
			}},
		})
		return
	}

	// 3. Set ID from path parameter
	req.TenantConfig.ID = id

	// 4. Call service layer for business logic
	updatedConfig, err := h.Service.UpdateTenantConfigWithValidation(context.Background(), &req.TenantConfig, c.GetHeader("X-Client-Id"))
	if err != nil {
		// Handle specific error types
		if err.Error() == "RECORD_NOT_FOUND" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Errors: []models.Error{{
					Code:        "RECORD_NOT_FOUND",
					Message:     "Record doesn't exist",
					Description: "No tenant config found with the given ID",
				}},
			})
			return
		}

		if err.Error() == "DOCUMENTS_REQUIRED" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Errors: []models.Error{{
					Code:        "DOCUMENTS_REQUIRED",
					Message:     "Documents array is required in update request",
					Description: "All existing documents must be included in the update request",
				}},
			})
			return
		}

		// Handle missing document error
		if len(err.Error()) > 15 && err.Error()[:15] == "MISSING_DOCUMENT" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Errors: []models.Error{{
					Code:        "MISSING_DOCUMENT",
					Message:     "Missing document in update request",
					Description: err.Error(),
				}},
			})
			return
		}

		// Handle document not found error
		if len(err.Error()) > 20 && err.Error()[:20] == "DOCUMENT_NOT_FOUND" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Errors: []models.Error{{
					Code:        "DOCUMENT_NOT_FOUND",
					Message:     "Document not found",
					Description: err.Error(),
				}},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "INTERNAL_ERROR",
				Message:     "Failed to update tenant config",
				Description: err.Error(),
			}},
		})
		return
	}

	// 5. Return success response
	resp := models.TenantConfigResponse{
		TenantConfigs: []models.TenantConfig{*updatedConfig},
	}
	c.JSON(http.StatusOK, resp)
}

func (h *TenantConfigHandler) DeleteTenantConfig(c *gin.Context) {
	id := c.Param("id")
	err := h.Service.DeleteTenantConfig(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "INTERNAL_ERROR",
				Message:     "Failed to delete tenant config",
				Description: err.Error(),
			}},
		})
		return
	}
	c.JSON(http.StatusNoContent, gin.H{})
}
