package service

import (
	"account/internal/enrichment"
	"account/internal/keycloak"
	"account/internal/models"
	"account/internal/notification"
	"account/internal/repository"
	"account/internal/validator"
	"context"
	"errors"
	"fmt"
	"log"
)

type TenantService struct {
	Repo               *repository.TenantRepository
	KeycloakClient     *keycloak.KeycloakClient
	NotificationClient *notification.Client
}

func NewTenantService(repo *repository.TenantRepository, keycloakClient *keycloak.KeycloakClient, notificationClient *notification.Client) *TenantService {
	return &TenantService{
		Repo:               repo,
		KeycloakClient:     keycloakClient,
		NotificationClient: notificationClient,
	}
}

func (s *TenantService) CreateTenant(ctx context.Context, tenant *models.Tenant) error {
	// 1. Create Keycloak realm first
	if err := s.createKeycloakRealmWithFullConfig(tenant); err != nil {
		log.Printf("Failed to create Keycloak realm for tenant %s: %v", tenant.Code, err)
		return fmt.Errorf("failed to create Keycloak realm: %v", err)
	}

	// 2. Only if Keycloak succeeds, create database entry
	if err := s.Repo.CreateTenant(ctx, tenant); err != nil {
		log.Printf("Failed to create tenant %s in database after Keycloak success: %v", tenant.Code, err)

		// Rollback: Delete the created Keycloak realm
		log.Printf("Rolling back: Deleting Keycloak realm %s due to database failure", tenant.Code)
		if rollbackErr := s.KeycloakClient.DeleteRealm(tenant.Code); rollbackErr != nil {
			log.Printf("WARNING: Failed to rollback Keycloak realm %s: %v", tenant.Code, rollbackErr)
			return fmt.Errorf("failed to create tenant in database: %v (rollback also failed: %v)", err, rollbackErr)
		}
		log.Printf("Successfully rolled back Keycloak realm %s", tenant.Code)

		return fmt.Errorf("failed to create tenant in database: %v", err)
	}

	// 3. Create notification template after successful realm and database creation
	if s.NotificationClient != nil {
		log.Printf("Creating notification template for tenant %s", tenant.Code)
		if err := s.NotificationClient.CreateSandboxEmailTemplate(tenant.Code, "https://digit-lts.digit.org/", tenant.Code); err != nil {
			log.Printf("WARNING: Failed to create notification template for tenant %s: %v", tenant.Code, err)
			// Don't fail the entire tenant creation for notification template failure
		} else {
			log.Printf("Successfully created notification template for tenant %s", tenant.Code)
		}
	}

	log.Printf("Successfully created tenant %s in both Keycloak and database", tenant.Code)
	return nil
}

func (s *TenantService) CreateTenantWithValidation(ctx context.Context, tenant *models.Tenant, clientID string) error {
	// 1. Validate tenant
	existing, err := s.Repo.ListTenants(ctx)
	if err != nil {
		return err
	}
	if err := validator.ValidateTenant(tenant, existing); err != nil {
		return err
	}

	// 2. Enrich tenant
	enrichment.EnrichTenant(tenant, clientID)

	// 3. Create tenant (includes Keycloak setup)
	return s.CreateTenant(ctx, tenant)
}

func (s *TenantService) createKeycloakRealmWithFullConfig(tenant *models.Tenant) error {
	log.Printf("Starting Keycloak realm creation for tenant %s", tenant.Code)

	// Use the enhanced realm creation with full configuration
	if err := s.KeycloakClient.CreateRealmWithFullConfig(
		tenant.Code,
		tenant.Email,
		tenant.Name,
		"999999999999", // Default mobile number
	); err != nil {
		log.Printf("Failed to create Keycloak realm with full config for tenant %s: %v", tenant.Code, err)
		return err
	}

	log.Printf("Successfully created Keycloak realm '%s' with full configuration for tenant %s", tenant.Code, tenant.Code)
	return nil
}

func (s *TenantService) UpdateTenant(ctx context.Context, updated *models.Tenant, clientID string) (*models.Tenant, error) {
	existing, err := s.Repo.GetTenantByID(ctx, updated.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("RECORD_NOT_FOUND")
	}
	// Only allow isActive and additionalAttributes to be updated
	existing.IsActive = updated.IsActive
	existing.AdditionalAttributes = updated.AdditionalAttributes
	enrichment.EnrichTenantUpdate(existing, clientID)
	if err := s.Repo.UpdateTenant(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *TenantService) ListTenants(ctx context.Context) ([]*models.Tenant, error) {
	return s.Repo.ListTenants(ctx)
}

func (s *TenantService) GetTenantByID(ctx context.Context, id string) (*models.Tenant, error) {
	return s.Repo.GetTenantByID(ctx, id)
}

func (s *TenantService) DeleteTenant(ctx context.Context, id string) error {
	return s.Repo.DeleteTenant(ctx, id)
}

func (s *TenantService) DeleteAccount(ctx context.Context, tenantCode string) error {
	log.Printf("Starting account deletion for tenant: %s", tenantCode)

	// 1. Get tenant by code to find the ID
	tenants, err := s.Repo.ListTenants(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tenants: %v", err)
	}

	var tenant *models.Tenant
	for _, t := range tenants {
		if t.Code == tenantCode {
			tenant = t
			break
		}
	}

	if tenant == nil {
		return fmt.Errorf("tenant with code %s not found", tenantCode)
	}

	// 2. Delete Keycloak realm first
	log.Printf("Deleting Keycloak realm: %s", tenantCode)
	if err := s.KeycloakClient.DeleteRealm(tenantCode); err != nil {
		log.Printf("Failed to delete Keycloak realm %s: %v", tenantCode, err)
		return fmt.Errorf("failed to delete Keycloak realm: %v", err)
	}
	log.Printf("Successfully deleted Keycloak realm: %s", tenantCode)

	// 3. Delete tenant from database (this will cascade delete tenant configs due to foreign key constraints)
	log.Printf("Deleting tenant from database: %s (ID: %s)", tenantCode, tenant.ID)
	if err := s.Repo.DeleteTenant(ctx, tenant.ID); err != nil {
		log.Printf("Failed to delete tenant %s from database: %v", tenantCode, err)
		return fmt.Errorf("failed to delete tenant from database: %v", err)
	}
	log.Printf("Successfully deleted tenant from database: %s", tenantCode)

	log.Printf("Successfully completed account deletion for tenant: %s", tenantCode)
	return nil
}
