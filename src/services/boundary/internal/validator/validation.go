package validator

import (
	"boundary/internal/models"
	"context"
	"encoding/json"
	"errors"
)

type BoundaryValidator struct {
	Repo BoundaryRepoChecker // interface for DB duplicate check
}

type BoundaryRepoChecker interface {
	ExistsByCode(ctx context.Context, tenantId, code string) (bool, error)
}

func (v *BoundaryValidator) ValidateBoundary(ctx context.Context, boundary *models.Boundary) error {
	// Required fields
	if boundary.TenantID == "" {
		return errors.New("Missing required field: tenantId")
	}
	if boundary.Code == "" {
		return errors.New("Missing required field: code")
	}
	// Geometry type and coordinates validation
	var geom map[string]interface{}
	if err := json.Unmarshal(boundary.Geometry, &geom); err != nil {
		return errors.New("Invalid geometry JSON")
	}
	typeStr, ok := geom["type"].(string)
	if !ok || !models.IsValidGeometryType(typeStr) {
		return errors.New("Invalid geometry type: Allowed types are Point, Polygon, MultiPolygon")
	}
	coords, ok := geom["coordinates"]
	if !ok {
		return errors.New("Missing coordinates in geometry")
	}
	if err := validateCoordinates(typeStr, coords); err != nil {
		return err
	}
	// Duplicate check
	if v.Repo != nil {
		exists, err := v.Repo.ExistsByCode(ctx, boundary.TenantID, boundary.Code)
		if err != nil {
			return err
		}
		if exists {
			return errors.New("Duplicate boundary code for tenant")
		}
	}
	return nil
}

func validateCoordinates(typeStr string, coords interface{}) error {
	switch typeStr {
	case "Point":
		arr, ok := coords.([]interface{})
		if !ok || len(arr) != 2 || !isNumber(arr[0]) || !isNumber(arr[1]) {
			return errors.New("Point geometry must have [x, y] coordinates")
		}
	case "Polygon":
		arr, ok := coords.([]interface{})
		if !ok || len(arr) == 0 {
			return errors.New("Polygon geometry must be an array of linear rings")
		}
		for _, ring := range arr {
			ringArr, ok := ring.([]interface{})
			if !ok || len(ringArr) < 4 {
				return errors.New("Each polygon ring must have at least 4 points")
			}
		}
	case "MultiPolygon":
		arr, ok := coords.([]interface{})
		if !ok || len(arr) == 0 {
			return errors.New("MultiPolygon geometry must be an array of polygons")
		}
		for _, poly := range arr {
			polyArr, ok := poly.([]interface{})
			if !ok || len(polyArr) == 0 {
				return errors.New("Each MultiPolygon must contain polygons")
			}
			for _, ring := range polyArr {
				ringArr, ok := ring.([]interface{})
				if !ok || len(ringArr) < 4 {
					return errors.New("Each polygon ring in MultiPolygon must have at least 4 points")
				}
			}
		}
	default:
		return errors.New("Unsupported geometry type")
	}
	return nil
}

func isNumber(v interface{}) bool {
	_, ok := v.(float64)
	return ok
}

// BoundaryHierarchy validation

type BoundaryHierarchyValidator struct {
	Repo HierarchyRepoChecker
}

type HierarchyRepoChecker interface {
	ExistsByType(ctx context.Context, tenantId, hierarchyType string) (bool, error)
}

func (v *BoundaryHierarchyValidator) ValidateHierarchy(ctx context.Context, hierarchy *models.BoundaryHierarchy) error {
	if hierarchy.TenantID == "" || hierarchy.HierarchyType == "" {
		return errors.New("Missing required fields in boundary hierarchy")
	}
	if v.Repo != nil {
		exists, err := v.Repo.ExistsByType(ctx, hierarchy.TenantID, hierarchy.HierarchyType)
		if err != nil {
			return err
		}
		if exists {
			return errors.New("Duplicate hierarchy type for tenant")
		}
	}
	return nil
}

// BoundaryRelationship validation

type BoundaryRelationshipValidator struct {
	Repo RelationshipRepoChecker
}

type RelationshipRepoChecker interface {
	ExistsByCode(ctx context.Context, tenantId, code, hierarchyType string) (bool, error)
	ParentExists(ctx context.Context, tenantId, parent, hierarchyType string) (bool, error)
	BoundaryExists(ctx context.Context, tenantId, boundaryCode string) (bool, error)
}

func (v *BoundaryRelationshipValidator) ValidateRelationship(ctx context.Context, rel *models.BoundaryRelationship) error {
	if rel.TenantID == "" || rel.Code == "" || rel.HierarchyType == "" {
		return errors.New("Missing required fields in boundary relationship")
	}
	if rel.BoundaryType == "" {
		return errors.New("Missing required field in boundary relationship: boundaryType")
	}
	if v.Repo != nil {
		// Check if boundary entity exists for the given code
		boundaryExists, err := v.Repo.BoundaryExists(ctx, rel.TenantID, rel.Code)
		if err != nil {
			return err
		}
		if !boundaryExists {
			return errors.New("Boundary with code '" + rel.Code + "' does not exist")
		}
		
		// Check for duplicate relationship
		exists, err := v.Repo.ExistsByCode(ctx, rel.TenantID, rel.Code, rel.HierarchyType)
		if err != nil {
			return err
		}
		if exists {
			return errors.New("Duplicate relationship code for tenant and hierarchy type")
		}
		
		// Check parent relationship exists (if specified)
		if rel.Parent != "" {
			parentExists, err := v.Repo.ParentExists(ctx, rel.TenantID, rel.Parent, rel.HierarchyType)
			if err != nil {
				return err
			}
			if !parentExists {
				return errors.New("Parent relationship does not exist")
			}
		}
	}
	return nil
}
