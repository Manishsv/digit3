package repository

import (
	"boundary/internal/config"
	relationshipmodels "boundary/internal/models"
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// BoundaryRelationshipRepositoryImpl implements BoundaryRelationshipRepository
type BoundaryRelationshipRepositoryImpl struct {
	db     *gorm.DB
	config *config.Config
}

// NewBoundaryRelationshipRepository creates a new boundary relationship repository
func NewBoundaryRelationshipRepository(db *gorm.DB, config *config.Config) *BoundaryRelationshipRepositoryImpl {
	return &BoundaryRelationshipRepositoryImpl{
		db:     db,
		config: config,
	}
}

// Create implements BoundaryRelationshipRepository.Create
func (r *BoundaryRelationshipRepositoryImpl) Create(ctx context.Context, request *relationshipmodels.BoundaryRelationshipRequest) error {
	parent := request.BoundaryRelationship.Parent
	var parentPath string

	if parent != "" && parent != "null" {
		// Check if parent exists and get its materialized path
		var parentRelationship relationshipmodels.BoundaryRelationship
		err := r.db.WithContext(ctx).Where("code = ? AND tenantid = ? AND hierarchytype = ?", parent, request.BoundaryRelationship.TenantID, request.BoundaryRelationship.HierarchyType).First(&parentRelationship).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("parent relationship with code '%s' does not exist", parent)
			}
			return fmt.Errorf("error checking parent existence: %v", err)
		}
		parentPath = parentRelationship.AncestralMaterializedPath
	}

	// Build ancestral materialized path
	request.BoundaryRelationship.AncestralMaterializedPath = r.buildMaterializedPath(parentPath, request.BoundaryRelationship.Code)

	return r.db.WithContext(ctx).Create(&request.BoundaryRelationship).Error
}

// Search implements BoundaryRelationshipRepository.Search
func (r *BoundaryRelationshipRepositoryImpl) Search(ctx context.Context, criteria *relationshipmodels.BoundaryRelationshipSearchCriteria) ([]relationshipmodels.BoundaryRelationship, error) {
	var relationships []relationshipmodels.BoundaryRelationship

	query := r.db.WithContext(ctx).Where("tenantid = ?", criteria.TenantID)

	if len(criteria.Codes) > 0 {
		query = query.Where("code IN ?", criteria.Codes)
	}

	if criteria.HierarchyType != "" {
		query = query.Where("hierarchytype = ?", criteria.HierarchyType)
	}

	if criteria.BoundaryType != "" {
		query = query.Where("boundarytype = ?", criteria.BoundaryType)
	}

	if criteria.Parent != "" {
		query = query.Where("parent = ?", criteria.Parent)
	}

	if criteria.Limit > 0 {
		query = query.Limit(criteria.Limit)
	}

	if criteria.Offset > 0 {
		query = query.Offset(criteria.Offset)
	}

	err := query.Find(&relationships).Error
	if err != nil {
		return nil, fmt.Errorf("error querying boundary relationships: %v", err)
	}

	return relationships, nil
}

// Update implements BoundaryRelationshipRepository.Update
func (r *BoundaryRelationshipRepositoryImpl) Update(ctx context.Context, request *relationshipmodels.BoundaryRelationshipRequest) error {
	// Get current relationship to check if parent changed
	current, err := r.GetByID(ctx, request.BoundaryRelationship.ID, request.BoundaryRelationship.TenantID)
	if err != nil {
		return err
	}
	if current == nil {
		return fmt.Errorf("boundary relationship not found")
	}

	// If parent changed, update materialized path
	if current.Parent != request.BoundaryRelationship.Parent {
		var parentPath string
		if request.BoundaryRelationship.Parent != "" && request.BoundaryRelationship.Parent != "null" {
			// Get new parent's materialized path
			var parentRelationship relationshipmodels.BoundaryRelationship
			err := r.db.WithContext(ctx).Where("code = ? AND tenantid = ? AND hierarchytype = ?", request.BoundaryRelationship.Parent, request.BoundaryRelationship.TenantID, request.BoundaryRelationship.HierarchyType).First(&parentRelationship).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					return fmt.Errorf("parent relationship with code '%s' does not exist", request.BoundaryRelationship.Parent)
				}
				return fmt.Errorf("error getting parent: %v", err)
			}
			parentPath = parentRelationship.AncestralMaterializedPath
		}

		// Update materialized path
		request.BoundaryRelationship.AncestralMaterializedPath = r.buildMaterializedPath(parentPath, request.BoundaryRelationship.Code)

		// Update all children's materialized paths
		err = r.updateChildrenMaterializedPaths(ctx, current.Code, current.AncestralMaterializedPath, request.BoundaryRelationship.AncestralMaterializedPath, request.BoundaryRelationship.TenantID, request.BoundaryRelationship.HierarchyType)
		if err != nil {
			return fmt.Errorf("error updating children paths: %v", err)
		}
	}

	return r.db.WithContext(ctx).Where("id = ? AND tenantid = ?", request.BoundaryRelationship.ID, request.BoundaryRelationship.TenantID).Updates(&request.BoundaryRelationship).Error
}

// GetByID fetches a boundary relationship by ID and tenantId
func (r *BoundaryRelationshipRepositoryImpl) GetByID(ctx context.Context, id, tenantId string) (*relationshipmodels.BoundaryRelationship, error) {
	var relationship relationshipmodels.BoundaryRelationship
	err := r.db.WithContext(ctx).Where("id = ? AND tenantid = ?", id, tenantId).First(&relationship).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &relationship, nil
}

func (r *BoundaryRelationshipRepositoryImpl) ExistsByCode(ctx context.Context, tenantId, code, hierarchyType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&relationshipmodels.BoundaryRelationship{}).Where("tenantid = ? AND code = ? AND hierarchytype = ?", tenantId, code, hierarchyType).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *BoundaryRelationshipRepositoryImpl) ParentExists(ctx context.Context, tenantId, parent, hierarchyType string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&relationshipmodels.BoundaryRelationship{}).Where("tenantid = ? AND code = ? AND hierarchytype = ?", tenantId, parent, hierarchyType).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// BoundaryExists checks if a boundary entity exists for the given code
func (r *BoundaryRelationshipRepositoryImpl) BoundaryExists(ctx context.Context, tenantId, boundaryCode string) (bool, error) {
	var count int64
	// Check in the boundary_v1 table (correct table name)
	err := r.db.WithContext(ctx).Table("boundary_v1").Where("tenantid = ? AND code = ?", tenantId, boundaryCode).Count(&count).Error
	if err != nil {
		fmt.Printf("Error checking boundary existence for tenantId=%s, code=%s: %v\n", tenantId, boundaryCode, err)
		return false, fmt.Errorf("error checking boundary existence: %v", err)
	}
	fmt.Printf("Boundary existence check: tenantId=%s, code=%s, count=%d\n", tenantId, boundaryCode, count)
	return count > 0, nil
}

// buildMaterializedPath builds the ancestral materialized path
func (r *BoundaryRelationshipRepositoryImpl) buildMaterializedPath(parentPath, code string) string {
	if parentPath == "" {
		return code
	}
	return parentPath + "|" + code
}

// updateChildrenMaterializedPaths updates materialized paths for all children when parent changes
func (r *BoundaryRelationshipRepositoryImpl) updateChildrenMaterializedPaths(ctx context.Context, nodeCode, oldPath, newPath, tenantId, hierarchyType string) error {
	// Find all children that have this node in their materialized path
	var children []relationshipmodels.BoundaryRelationship
	err := r.db.WithContext(ctx).Where("tenantid = ? AND hierarchytype = ? AND ancestralmaterializedpath LIKE ?", tenantId, hierarchyType, oldPath+"|%").Find(&children).Error
	if err != nil {
		return err
	}

	// Update each child's materialized path
	for _, child := range children {
		updatedPath := strings.Replace(child.AncestralMaterializedPath, oldPath, newPath, 1)
		err = r.db.WithContext(ctx).Model(&child).Where("id = ?", child.ID).Update("ancestralmaterializedpath", updatedPath).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// SearchWithMaterializedPath searches using materialized path for efficient hierarchy queries
func (r *BoundaryRelationshipRepositoryImpl) SearchWithMaterializedPath(ctx context.Context, criteria *relationshipmodels.BoundaryRelationshipSearchCriteria) ([]relationshipmodels.BoundaryRelationship, error) {
	var relationships []relationshipmodels.BoundaryRelationship

	query := r.db.WithContext(ctx).Where("tenantid = ?", criteria.TenantID)

	if criteria.HierarchyType != "" {
		query = query.Where("hierarchytype = ?", criteria.HierarchyType)
	}

	if criteria.BoundaryType != "" {
		query = query.Where("boundarytype = ?", criteria.BoundaryType)
	}

	if criteria.Parent != "" {
		query = query.Where("parent = ?", criteria.Parent)
	}

	// Handle isSearchForRootNode flag (Java-like implementation)
	if criteria.IsSearchForRootNode {
		query = query.Where("parent IS NULL OR parent = ''")
	}

	// Handle currentBoundaryCodes for finding children (Java-like implementation)
	if len(criteria.CurrentBoundaryCodes) > 0 {
		// This mimics Java's: ARRAY[currentBoundaryCodes] && string_to_array(ancestralmaterializedpath, '|')
		placeholders := make([]string, len(criteria.CurrentBoundaryCodes))
		args := make([]interface{}, len(criteria.CurrentBoundaryCodes))
		for i, code := range criteria.CurrentBoundaryCodes {
			placeholders[i] = "?"
			args[i] = code
		}

		// PostgreSQL array overlap operator to find boundaries that have any of the current boundary codes in their path
		arrayQuery := fmt.Sprintf("ARRAY[%s]::text[] && string_to_array(ancestralmaterializedpath, '|')", strings.Join(placeholders, ","))
		query = query.Where(arrayQuery, args...)
	}

	// Handle codes filter with PostgreSQL array operations (similar to Java implementation)
	if len(criteria.Codes) > 0 && !criteria.IsSearchForRootNode {
		if criteria.IncludeChildren {
			// Use PostgreSQL array operations to find children efficiently
			// This mimics the Java implementation: ARRAY[codes] && string_to_array(ancestralmaterializedpath, '|')
			placeholders := make([]string, len(criteria.Codes))
			args := make([]interface{}, len(criteria.Codes))
			for i, code := range criteria.Codes {
				placeholders[i] = "?"
				args[i] = code
			}

			// PostgreSQL array overlap operator to check if any code exists in the materialized path
			arrayQuery := fmt.Sprintf("ARRAY[%s]::text[] && string_to_array(ancestralmaterializedpath, '|')", strings.Join(placeholders, ","))
			query = query.Where(arrayQuery, args...)
		} else if criteria.IncludeParents {
			// For parents, we need to get the materialized paths first, then find all ancestors
			// This is more complex and requires a subquery approach
			subquery := r.db.WithContext(ctx).Model(&relationshipmodels.BoundaryRelationship{}).
				Select("unnest(string_to_array(ancestralmaterializedpath, '|')) as ancestor_code").
				Where("code IN ? AND tenantid = ? AND hierarchytype = ?", criteria.Codes, criteria.TenantID, criteria.HierarchyType)

			query = query.Where("code IN (?)", subquery)
		} else {
			// Simple code filter
			query = query.Where("code IN ?", criteria.Codes)
		}
	}

	if criteria.Limit > 0 {
		query = query.Limit(criteria.Limit)
	}

	if criteria.Offset > 0 {
		query = query.Offset(criteria.Offset)
	}

	// Order by materialized path for hierarchical ordering, then by creation time
	err := query.Order("ancestralmaterializedpath, createdtime desc").Find(&relationships).Error
	if err != nil {
		return nil, fmt.Errorf("error querying boundary relationships: %v", err)
	}

	return relationships, nil
}
