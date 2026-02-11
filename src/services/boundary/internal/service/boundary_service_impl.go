package service

import (
	commonmodels "boundary/internal/common/models"
	"boundary/internal/models"
	"boundary/internal/repository"
	"boundary/internal/validator"
	"boundary/pkg/cache"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BoundaryServiceImpl implements the BoundaryService interface
type BoundaryServiceImpl struct {
	repo  repository.BoundaryRepository
	cache cache.Cache
}

// NewBoundaryService creates a new boundary service
func NewBoundaryService(repo repository.BoundaryRepository, cache cache.Cache) BoundaryService {
	return &BoundaryServiceImpl{
		repo:  repo,
		cache: cache,
	}
}

// Create implements BoundaryService.Create
func (s *BoundaryServiceImpl) Create(ctx context.Context, request *models.BoundaryRequest, tenantID, clientID string) error {
	epoch := time.Now().UnixMilli()
	validator := &validator.BoundaryValidator{Repo: s.repo}
	var createdCodes []string
	for i := range request.Boundary {
		request.Boundary[i].TenantID = tenantID
		if err := validator.ValidateBoundary(ctx, &request.Boundary[i]); err != nil {
			return err
		}
	}
	for i := range request.Boundary {
		request.Boundary[i].ID = uuid.New().String()
		request.Boundary[i].AuditDetails = &commonmodels.AuditDetails{
			CreatedBy:        clientID,
			LastModifiedBy:   clientID,
			CreatedTime:      epoch,
			LastModifiedTime: epoch,
		}
		// Track codes for cache invalidation
		createdCodes = append(createdCodes, request.Boundary[i].Code)
	}
	err := s.repo.Create(ctx, request)
	if err != nil {
		return err
	}
	// Invalidate all search caches that might contain these boundary codes
	s.invalidateSearchCaches(ctx, tenantID, createdCodes)
	return nil
}

// Search implements BoundaryService.Search
func (s *BoundaryServiceImpl) Search(ctx context.Context, criteria *models.BoundarySearchCriteria) ([]models.Boundary, error) {
	cacheKey := criteria.TenantID + ":boundary:search:" + strings.Join(criteria.Codes, ",")
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		var result []models.Boundary
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

// Update implements BoundaryService.Update
func (s *BoundaryServiceImpl) Update(ctx context.Context, request *models.BoundaryRequest, tenantID, clientID string) error {
	epoch := time.Now().UnixMilli()
	var updatedCodes []string
	for i := range request.Boundary {
		request.Boundary[i].TenantID = tenantID
		// Fetch existing record
		existing, err := s.repo.GetByID(ctx, request.Boundary[i].ID, tenantID)
		if err != nil {
			return err
		}
		if existing == nil {
			return fmt.Errorf("boundary with id %s does not exist", request.Boundary[i].ID)
		}
		if request.Boundary[i].AuditDetails == nil {
			request.Boundary[i].AuditDetails = &commonmodels.AuditDetails{}
		}
		// Preserve createdBy and createdTime from DB
		request.Boundary[i].AuditDetails.CreatedBy = existing.AuditDetails.CreatedBy
		request.Boundary[i].AuditDetails.CreatedTime = existing.AuditDetails.CreatedTime
		// Set last modified fields
		request.Boundary[i].AuditDetails.LastModifiedBy = clientID
		request.Boundary[i].AuditDetails.LastModifiedTime = epoch
		// Track codes for cache invalidation
		updatedCodes = append(updatedCodes, request.Boundary[i].Code)
	}
	err := s.repo.Update(ctx, request)
	if err != nil {
		return err
	}
	// Invalidate all search caches that might contain these boundary codes
	s.invalidateSearchCaches(ctx, tenantID, updatedCodes)
	return nil
}

// invalidateSearchCaches invalidates all search cache entries that might contain the given boundary codes
func (s *BoundaryServiceImpl) invalidateSearchCaches(ctx context.Context, tenantID string, boundaryCodes []string) {
	// Method 1: If cache supports pattern-based deletion (Redis)
	if redisCache, ok := s.cache.(interface{ DeletePattern(context.Context, string) error }); ok {
		for _, code := range boundaryCodes {
			// Delete all search caches for this tenant that might contain the boundary code
			pattern := tenantID + ":boundary:search:*" + code + "*"
			_ = redisCache.DeletePattern(ctx, pattern)
		}
		return
	}
	
	// Method 2: Fallback for in-memory cache - clear all search caches for tenant
	if memCache, ok := s.cache.(interface{ DeleteByPrefix(context.Context, string) error }); ok {
		prefix := tenantID + ":boundary:search:"
		_ = memCache.DeleteByPrefix(ctx, prefix)
		return
	}
	
	// Method 3: Basic fallback - delete specific known patterns
	for _, code := range boundaryCodes {
		// Delete single code cache
		singleCodeKey := tenantID + ":boundary:search:" + code
		_ = s.cache.Delete(ctx, singleCodeKey)
	}
	
	// Delete the exact combination cache
	cacheKey := tenantID + ":boundary:search:" + strings.Join(boundaryCodes, ",")
	_ = s.cache.Delete(ctx, cacheKey)
}
