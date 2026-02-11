package repository

import (
	"gin/models"

	"gorm.io/gorm"
)

// ArtifactRepository defines the interface for artifact persistence
type ArtifactRepository interface {
	SaveAll(artifacts []models.ArtifactEntity) ([]models.ArtifactEntity, error)
	FindByFileStoreIdAndTenantId(fileStoreId, tenantId string) (*models.ArtifactEntity, error)
	FindByTagAndTenantId(tag, tenantId string) ([]models.ArtifactEntity, error)
	UpdateContentType(fileStoreId, tenantId, contentType string) error
	DeleteByFileStoreIdAndTenantId(fileStoreId, tenantId string) error
	CreateDocumentCategory(doc models.DocumentCategory, tenantId string) (models.DocumentCategory, error)
	SearchDocumentCategories(docType string, docCode string, isSensitive string, tenantId string) ([]models.DocumentCategory, error)
	UpdateDocumentCategory(doc models.DocumentCategory, docCode string, tenantId string) error
	DeleteDocumentCategory(docCode string, tenantId string)
}

// PostgresArtifactRepository implements ArtifactRepository using PostgreSQL
type PostgresArtifactRepository struct {
	db *gorm.DB
}

// NewPostgresArtifactRepository creates a new PostgreSQL repository
func NewPostgresArtifactRepository(db *gorm.DB) ArtifactRepository {
	return &PostgresArtifactRepository{db: db}
}

// SaveAll saves multiple artifacts to the database
func (r *PostgresArtifactRepository) SaveAll(artifacts []models.ArtifactEntity) ([]models.ArtifactEntity, error) {
	result := r.db.Create(&artifacts)
	if result.Error != nil {
		return nil, result.Error
	}
	return artifacts, nil
}

// FindByFileStoreIdAndTenantId finds an artifact by fileStoreId and tenantId
func (r *PostgresArtifactRepository) FindByFileStoreIdAndTenantId(fileStoreId, tenantId string) (*models.ArtifactEntity, error) {
	var artifact models.ArtifactEntity
	result := r.db.Where("filestoreid = ? AND tenantid = ?", fileStoreId, tenantId).First(&artifact)
	if result.Error != nil {
		return nil, result.Error
	}
	return &artifact, nil
}

// FindByTagAndTenantId finds artifacts by tag and tenantId
func (r *PostgresArtifactRepository) FindByTagAndTenantId(tag, tenantId string) ([]models.ArtifactEntity, error) {
	var artifacts []models.ArtifactEntity
	result := r.db.Where("tag = ? AND tenantid = ?", tag, tenantId).Find(&artifacts)
	if result.Error != nil {
		return nil, result.Error
	}
	return artifacts, nil
}

func (r *PostgresArtifactRepository) UpdateContentType(fileStoreId, tenantId, contentType string) error {
	result := r.db.Model(&models.ArtifactEntity{}).
		Where("filestoreid = ? AND tenantid = ?", fileStoreId, tenantId).
		Update("contenttype", contentType)
	return result.Error
}

func (r *PostgresArtifactRepository) DeleteByFileStoreIdAndTenantId(fileStoreId, tenantId string) error {
	result := r.db.Where("filestoreid = ? AND tenantid = ?", fileStoreId, tenantId).Delete(&models.ArtifactEntity{})
	return result.Error
}

func (r *PostgresArtifactRepository) CreateDocumentCategory(doc models.DocumentCategory, tenantId string) (models.DocumentCategory, error) {
	result := r.db.Create(&doc)
	if result.Error != nil {
		return models.DocumentCategory{}, result.Error
	}
	return doc, nil
}

func (r *PostgresArtifactRepository) SearchDocumentCategories(docType string, docCode string, isSensitive string, tenantId string) ([]models.DocumentCategory, error) {
	var docs []models.DocumentCategory
	query := r.db.Model(&models.DocumentCategory{})

	if docType != "" {
		query = query.Where("type = ?", docType)
	}
	if docCode != "" {
		query = query.Where("code = ?", docCode)
	}
	if isSensitive != "" {
		query = query.Where("isSensitive = ?", isSensitive)
	}
	if tenantId != "" {
		query = query.Where("tenantId = ?", tenantId)
	}

	result := query.Find(&docs)
	return docs, result.Error
}

func (r *PostgresArtifactRepository) DeleteDocumentCategory(docCode string, tenantId string) {
	r.db.Where("code = ? AND tenantId = ?", docCode, tenantId).Delete(&models.DocumentCategory{})
}

func (r *PostgresArtifactRepository) UpdateDocumentCategory(doc models.DocumentCategory, docCode string, tenantId string) error {
	result := r.db.Model(&models.DocumentCategory{}).
		Where("code = ? AND tenantId = ?", docCode, tenantId).
		Updates(doc)
	return result.Error
}
