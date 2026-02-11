package repository

import (
	"boundary/internal/config"
	hierarchymodels "boundary/internal/models"
	"context"
	"fmt"

	"gorm.io/gorm"
)

// BoundaryHierarchyRepositoryImpl implements BoundaryHierarchyRepository
type BoundaryHierarchyRepositoryImpl struct {
	db     *gorm.DB
	config *config.Config
}

// NewBoundaryHierarchyRepository creates a new boundary hierarchy repository
func NewBoundaryHierarchyRepository(db *gorm.DB, config *config.Config) *BoundaryHierarchyRepositoryImpl {
	return &BoundaryHierarchyRepositoryImpl{
		db:     db,
		config: config,
	}
}

// Create implements BoundaryHierarchyRepository.Create
func (r *BoundaryHierarchyRepositoryImpl) Create(ctx context.Context, request *hierarchymodels.BoundaryHierarchyRequest) error {
	return r.db.WithContext(ctx).Create(&request.BoundaryHierarchy).Error
}

// Search implements BoundaryHierarchyRepository.Search
func (r *BoundaryHierarchyRepositoryImpl) Search(ctx context.Context, criteria *hierarchymodels.BoundaryHierarchySearchCriteria) ([]hierarchymodels.BoundaryHierarchy, error) {
	var hierarchies []hierarchymodels.BoundaryHierarchy
	err := r.db.WithContext(ctx).Where("tenantid = ? AND hierarchytype = ?", criteria.TenantID, criteria.HierarchyType).Find(&hierarchies).Error
	if err != nil {
		return nil, fmt.Errorf("error querying boundary hierarchies: %v", err)
	}
	return hierarchies, nil
}

// Update implements BoundaryHierarchyRepository.Update
func (r *BoundaryHierarchyRepositoryImpl) Update(ctx context.Context, request *hierarchymodels.BoundaryHierarchyRequest) error {
	return r.db.WithContext(ctx).Where("id = ? AND tenantid = ? AND hierarchytype = ?",
		request.BoundaryHierarchy.ID,
		request.BoundaryHierarchy.TenantID,
		request.BoundaryHierarchy.HierarchyType).Updates(&request.BoundaryHierarchy).Error
}

func (r *BoundaryHierarchyRepositoryImpl) ExistsByType(ctx context.Context, tenantId, hierarchyType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&hierarchymodels.BoundaryHierarchy{}).Where("tenantid = ? AND hierarchytype = ?", tenantId, hierarchyType).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
