package handlers

import (
	"account/internal/models"
	"account/internal/service"
	"account/internal/validator"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TenantHandler struct {
	Service *service.TenantService
}

func NewTenantHandler(svc *service.TenantService) *TenantHandler {
	return &TenantHandler{Service: svc}
}

func (h *TenantHandler) CreateTenant(c *gin.Context) {
	// 1. Parse request body
	var req models.TenantRequest
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
	err := h.Service.CreateTenantWithValidation(context.Background(), &req.Tenant, c.GetHeader("X-Client-Id"))
	if err != nil {
		// Handle validation errors
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
				Message:     "Failed to create tenant",
				Description: err.Error(),
			}},
		})
		return
	}

	// 3. Return success response
	resp := models.TenantResponse{
		Tenants: []models.Tenant{req.Tenant},
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *TenantHandler) GetTenantByID(c *gin.Context) {
	id := c.Param("id")
	t, err := h.Service.GetTenantByID(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "INTERNAL_ERROR",
				Message:     "Failed to get tenant",
				Description: err.Error(),
			}},
		})
		return
	}
	if t == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "NOT_FOUND",
				Message:     "Tenant not found",
				Description: "No tenant found with the given ID",
			}},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"Tenant": t,
	})
}

func (h *TenantHandler) ListTenants(c *gin.Context) {
	// Get query parameters for filtering
	code := c.Query("code")
	name := c.Query("name")

	tenants, err := h.Service.ListTenants(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "INTERNAL_ERROR",
				Message:     "Failed to list tenants",
				Description: err.Error(),
			}},
		})
		return
	}
	var filtered []models.Tenant
	for _, t := range tenants {
		if (code == "" || t.Code == code) && (name == "" || t.Name == name) {
			filtered = append(filtered, *t)
		}
	}
	resp := models.TenantResponse{
		Tenants: filtered,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	// Get ID from path parameter
	id := c.Param("id")

	var req models.TenantRequest
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

	// Set the ID from path parameter
	req.Tenant.ID = id

	updated, err := h.Service.UpdateTenant(context.Background(), &req.Tenant, c.GetHeader("X-Client-Id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "INTERNAL_ERROR",
				Message:     "Failed to update tenant",
				Description: err.Error(),
			}},
		})
		return
	}
	resp := models.TenantResponse{
		Tenants: []models.Tenant{*updated},
	}
	c.JSON(http.StatusOK, resp)
}

func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	id := c.Param("id")
	err := h.Service.DeleteTenant(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "INTERNAL_ERROR",
				Message:     "Failed to delete tenant",
				Description: err.Error(),
			}},
		})
		return
	}
	c.JSON(http.StatusNoContent, gin.H{})
}

func (h *TenantHandler) DeleteAccount(c *gin.Context) {
	// Get tenant code from query parameter
	tenantCode := c.Query("tenantCode")
	if tenantCode == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "BAD_REQUEST",
				Message:     "Missing required parameter",
				Description: "tenantCode query parameter is required",
			}},
		})
		return
	}

	// Call service to delete account (realm, tenant, and config)
	err := h.Service.DeleteAccount(context.Background(), tenantCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Errors: []models.Error{{
				Code:        "INTERNAL_ERROR",
				Message:     "Failed to delete account",
				Description: err.Error(),
			}},
		})
		return
	}

	c.JSON(http.StatusNoContent, gin.H{
		"message": "Account, realm, and configuration deleted successfully",
	})
}
