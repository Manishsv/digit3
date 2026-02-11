package repository

import (
	"template-config/internal/models"

	"gorm.io/gorm"
)

type TemplateConfigRepository struct {
	db *gorm.DB
}

func NewTemplateConfigRepository(db *gorm.DB) *TemplateConfigRepository {
	return &TemplateConfigRepository{db: db}
}

func (r *TemplateConfigRepository) Create(config *models.TemplateConfigDB) error {
	return r.db.Create(config).Error
}

func (r *TemplateConfigRepository) GetLatestTemplateConfig(templateID, tenantID string) (*models.TemplateConfigDB, error) {
	var config models.TemplateConfigDB
	err := r.db.
		Where("tenantid = ? AND templateid = ?", tenantID, templateID).
		Order("version DESC").
		First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *TemplateConfigRepository) GetByTemplateIDAndVersion(templateID, tenantID string, version int) (*models.TemplateConfigDB, error) {
	var config models.TemplateConfigDB
	err := r.db.
		Where("tenantid = ? AND templateid = ? AND version = ?", tenantID, templateID, version).
		First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *TemplateConfigRepository) Search(search *models.TemplateConfigSearch) ([]models.TemplateConfigDB, error) {
	var configs []models.TemplateConfigDB

	// Base query with tenant filter (mandatory)
	baseQuery := r.db.Model(&models.TemplateConfigDB{}).
		Where("template_config.tenantid = ?", search.TenantID)

	// IDs filter (applies to all cases)
	if len(search.IDs) > 0 {
		baseQuery = baseQuery.Where("template_config.id IN ?", search.IDs)
	}

	// Version explicitly provided
	if search.VersionInt > 0 {
		if search.TemplateID != "" {
			baseQuery = baseQuery.Where("template_config.templateid = ?", search.TemplateID)
		}
		baseQuery = baseQuery.Where("template_config.version = ?", search.VersionInt)
		return fetch(baseQuery, &configs)
	}

	// Case 1: tenantId + templateId (no version, no IDs)
	if search.TemplateID != "" && len(search.IDs) == 0 {
		sub := r.db.
			Table("template_config").
			Select("templateid, MAX(version) as version").
			Where("tenantid = ?", search.TenantID).
			Group("templateid")

		baseQuery = baseQuery.
			Joins(`JOIN (?) t2 
                ON template_config.templateid = t2.templateid 
               AND template_config.version = t2.version`, sub).
			Where("template_config.templateid = ?", search.TemplateID)

		return fetch(baseQuery, &configs)
	}

	// Case 2: tenantId only (latest version of each template)
	if search.TemplateID == "" && len(search.IDs) == 0 {
		sub := r.db.
			Table("template_config").
			Select("templateid, MAX(version) as version").
			Where("tenantid = ?", search.TenantID).
			Group("templateid")

		baseQuery = baseQuery.
			Joins(`JOIN (?) t2 
                ON template_config.templateid = t2.templateid 
               AND template_config.version = t2.version`, sub)

		return fetch(baseQuery, &configs)
	}

	// Case 3: tenantId + ids (+ templateId optional)
	if search.TemplateID != "" {
		baseQuery = baseQuery.Where("template_config.templateid = ?", search.TemplateID)
	}

	return fetch(baseQuery, &configs)
}

func fetch(tx *gorm.DB, configs *[]models.TemplateConfigDB) ([]models.TemplateConfigDB, error) {
	err := tx.Find(configs).Error
	return *configs, err
}

func (r *TemplateConfigRepository) Delete(templateID, tenantID string, version int) error {
	return r.db.Where("tenantid = ? AND templateid = ? AND version = ?", tenantID, templateID, version).Delete(&models.TemplateConfigDB{}).Error
}
