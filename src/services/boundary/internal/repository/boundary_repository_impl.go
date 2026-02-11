package repository

import (
	"boundary/internal/config"
	boundarymodels "boundary/internal/models"
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// BoundaryRepositoryImpl implements BoundaryRepository
type BoundaryRepositoryImpl struct {
	db     *gorm.DB
	config *config.Config
}

// NewBoundaryRepository creates a new boundary repository
func NewBoundaryRepository(db *gorm.DB, config *config.Config) *BoundaryRepositoryImpl {
	return &BoundaryRepositoryImpl{
		db:     db,
		config: config,
	}
}

// Create implements BoundaryRepository.Create
func (r *BoundaryRepositoryImpl) Create(ctx context.Context, request *boundarymodels.BoundaryRequest) error {
	tracer := otel.Tracer("boundary-repository")
	_, span := tracer.Start(ctx, "db.boundary.create")
	defer span.End()

	start := time.Now()
	span.SetAttributes(
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "boundary"),
		attribute.Int("boundary.count", len(request.Boundary)),
	)

	err := r.db.WithContext(ctx).Create(&request.Boundary).Error
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create boundaries")
		return err
	}

	span.SetStatus(codes.Ok, "Boundaries created successfully")
	return nil
}

// Search implements BoundaryRepository.Search
func (r *BoundaryRepositoryImpl) Search(ctx context.Context, criteria *boundarymodels.BoundarySearchCriteria) ([]boundarymodels.Boundary, error) {
	tracer := otel.Tracer("boundary-repository")
	_, span := tracer.Start(ctx, "db.boundary.search")
	defer span.End()

	start := time.Now()
	var boundaries []boundarymodels.Boundary

	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "boundary"),
		attribute.String("tenant.id", criteria.TenantID),
		attribute.StringSlice("boundary.codes", criteria.Codes),
		attribute.Int("boundary.limit", criteria.Limit),
		attribute.Int("boundary.offset", criteria.Offset),
	)

	query := r.db.WithContext(ctx).Where("tenantid = ?", criteria.TenantID)

	if len(criteria.Codes) > 0 {
		query = query.Where("code IN ?", criteria.Codes)
	}

	if criteria.Limit > 0 {
		query = query.Limit(criteria.Limit)
	}

	if criteria.Offset > 0 {
		query = query.Offset(criteria.Offset)
	}

	err := query.Find(&boundaries).Error
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
		attribute.Int("boundary.found_count", len(boundaries)),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to search boundaries")
		return nil, fmt.Errorf("error querying boundaries: %v", err)
	}

	span.SetStatus(codes.Ok, "Boundaries searched successfully")
	return boundaries, nil
}

// Update implements BoundaryRepository.Update
func (r *BoundaryRepositoryImpl) Update(ctx context.Context, request *boundarymodels.BoundaryRequest) error {
	tracer := otel.Tracer("boundary-repository")
	_, span := tracer.Start(ctx, "db.boundary.update")
	defer span.End()

	start := time.Now()
	span.SetAttributes(
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.table", "boundary"),
		attribute.Int("boundary.count", len(request.Boundary)),
	)

	for _, boundary := range request.Boundary {
		err := r.db.WithContext(ctx).Where("id = ? AND tenantid = ?", boundary.ID, boundary.TenantID).Updates(&boundary).Error
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to update boundary")
			return err
		}
	}

	duration := time.Since(start)
	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	)

	span.SetStatus(codes.Ok, "Boundaries updated successfully")
	return nil
}

// GetByID fetches a boundary by ID and tenantId
func (r *BoundaryRepositoryImpl) GetByID(ctx context.Context, id, tenantId string) (*boundarymodels.Boundary, error) {
	tracer := otel.Tracer("boundary-repository")
	_, span := tracer.Start(ctx, "db.boundary.get_by_id")
	defer span.End()

	start := time.Now()
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "boundary"),
		attribute.String("boundary.id", id),
		attribute.String("tenant.id", tenantId),
	)

	var boundary boundarymodels.Boundary
	err := r.db.WithContext(ctx).Where("id = ? AND tenantid = ?", id, tenantId).First(&boundary).Error
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			span.SetStatus(codes.Ok, "Boundary not found")
			return nil, nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get boundary by ID")
		return nil, err
	}

	span.SetStatus(codes.Ok, "Boundary retrieved successfully")
	return &boundary, nil
}

func (r *BoundaryRepositoryImpl) ExistsByCode(ctx context.Context, tenantId, code string) (bool, error) {
	tracer := otel.Tracer("boundary-repository")
	_, span := tracer.Start(ctx, "db.boundary.exists_by_code")
	defer span.End()

	start := time.Now()
	span.SetAttributes(
		attribute.String("db.operation", "COUNT"),
		attribute.String("db.table", "boundary"),
		attribute.String("boundary.code", code),
		attribute.String("tenant.id", tenantId),
	)

	var count int64
	err := r.db.WithContext(ctx).Model(&boundarymodels.Boundary{}).Where("tenantid = ? AND code = ?", tenantId, code).Count(&count).Error
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
		attribute.Int64("boundary.count", count),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to check boundary existence")
		return false, err
	}

	exists := count > 0
	span.SetAttributes(attribute.Bool("boundary.exists", exists))
	span.SetStatus(codes.Ok, "Boundary existence check completed")
	return exists, nil
}
