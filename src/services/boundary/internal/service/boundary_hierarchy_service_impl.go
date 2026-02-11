package service

import (
	commonmodels "boundary/internal/common/models"
	"boundary/internal/models"
	"boundary/internal/repository"
	"boundary/internal/validator"
	"boundary/pkg/cache"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BoundaryHierarchyServiceImpl implements BoundaryHierarchyService
type BoundaryHierarchyServiceImpl struct {
	repo  repository.BoundaryHierarchyRepository
	cache cache.Cache
}

// NewBoundaryHierarchyService creates a new boundary hierarchy service
func NewBoundaryHierarchyService(repo repository.BoundaryHierarchyRepository, cache cache.Cache) BoundaryHierarchyService {
	return &BoundaryHierarchyServiceImpl{
		repo:  repo,
		cache: cache,
	}
}

// Create implements BoundaryHierarchyService.Create
func (s *BoundaryHierarchyServiceImpl) Create(ctx context.Context, request *models.BoundaryHierarchyRequest, tenantID, clientID string) error {
	epoch := time.Now().UnixMilli()
	validator := &validator.BoundaryHierarchyValidator{Repo: s.repo}
	request.BoundaryHierarchy.TenantID = tenantID
	if err := validator.ValidateHierarchy(ctx, &request.BoundaryHierarchy); err != nil {
		return err
	}
	request.BoundaryHierarchy.ID = uuid.New().String()
	request.BoundaryHierarchy.AuditDetails = &commonmodels.AuditDetails{
		CreatedBy:        clientID,
		LastModifiedBy:   clientID,
		CreatedTime:      epoch,
		LastModifiedTime: epoch,
	}
	return s.repo.Create(ctx, request)
}

// Search implements BoundaryHierarchyService.Search
func (s *BoundaryHierarchyServiceImpl) Search(ctx context.Context, criteria *models.BoundaryHierarchySearchCriteria) ([]models.BoundaryHierarchy, error) {
	cacheKey := criteria.TenantID + ":hierarchy:search:" + criteria.HierarchyType
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		var result []models.BoundaryHierarchy
		if err := json.Unmarshal([]byte(cached.(string)), &result); err == nil {
			return result, nil
		}
	}
	result, err := s.repo.Search(ctx, criteria)
	if err != nil {
		return nil, err
	}
	if b, err := json.Marshal(result); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(b))
	}
	return result, nil
}

// Update implements BoundaryHierarchyService.Update
func (s *BoundaryHierarchyServiceImpl) Update(ctx context.Context, request *models.BoundaryHierarchyRequest, tenantID, clientID string) error {
	epoch := time.Now().UnixMilli()
	request.BoundaryHierarchy.TenantID = tenantID
	if request.BoundaryHierarchy.AuditDetails == nil {
		request.BoundaryHierarchy.AuditDetails = &commonmodels.AuditDetails{}
	}
	request.BoundaryHierarchy.AuditDetails.LastModifiedBy = clientID
	request.BoundaryHierarchy.AuditDetails.LastModifiedTime = epoch
	err := s.repo.Update(ctx, request)
	if err != nil {
		return err
	}
	// Invalidate cache for this hierarchy type
	cacheKey := tenantID + ":hierarchy:search:" + request.BoundaryHierarchy.HierarchyType
	_ = s.cache.Delete(ctx, cacheKey)
	return nil
}
