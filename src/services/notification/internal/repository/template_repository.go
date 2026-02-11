package repository

import (
	"notification/internal/models"

	"gorm.io/gorm"
)

type TemplateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) Create(template *models.TemplateDB) error {
	return r.db.Create(template).Error
}

func (r *TemplateRepository) GetLatestTemplate(templateID, tenantID string) (*models.TemplateDB, error) {
	var template models.TemplateDB
	err := r.db.
		Where("tenantid = ? AND templateid = ?", tenantID, templateID).
		Order("version DESC").
		First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *TemplateRepository) GetByTemplateIDAndVersion(templateID, tenantID string, version int) (*models.TemplateDB, error) {
	var config models.TemplateDB
	err := r.db.
		Where("tenantid = ? AND templateid = ? AND version = ?", tenantID, templateID, version).
		First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *TemplateRepository) Search(search *models.TemplateSearch) ([]models.TemplateDB, error) {
	var configs []models.TemplateDB

	// Base query with tenant filter (mandatory)
	baseQuery := r.db.Model(&models.TemplateDB{}).
		Where("notification_template.tenantid = ?", search.TenantID)

	// IDs filter (applies to all cases)
	if len(search.IDs) > 0 {
		baseQuery = baseQuery.Where("notification_template.id IN ?", search.IDs)
	}

	// Optional type filter
	if search.Type != "" {
		baseQuery = baseQuery.Where("notification_template.type = ?", search.Type)
	}

	// Optional isHTML filter (only meaningful if type=EMAIL)
	if search.IsHTML {
		baseQuery = baseQuery.Where("notification_template.ishtml = ?", true)
	}

	// Version explicitly provided
	if search.VersionInt > 0 {
		if search.TemplateID != "" {
			baseQuery = baseQuery.Where("notification_template.templateid = ?", search.TemplateID)
		}
		baseQuery = baseQuery.Where("notification_template.version = ?", search.VersionInt)
		return fetch(baseQuery, &configs)
	}

	// Case 1: tenantId + templateId (no version, no IDs)
	if search.TemplateID != "" && len(search.IDs) == 0 {
		sub := r.db.
			Table("notification_template").
			Select("templateid, MAX(version) as version").
			Where("tenantid = ?", search.TenantID).
			Group("templateid")

		baseQuery = baseQuery.
			Joins(`JOIN (?) t2 
                ON notification_template.templateid = t2.templateid 
               AND notification_template.version = t2.version`, sub).
			Where("notification_template.templateid = ?", search.TemplateID)

		return fetch(baseQuery, &configs)
	}

	// Case 2: tenantId only (latest version of each template)
	if search.TemplateID == "" && len(search.IDs) == 0 {
		sub := r.db.
			Table("notification_template").
			Select("templateid, MAX(version) as version").
			Where("tenantid = ?", search.TenantID).
			Group("templateid")

		baseQuery = baseQuery.
			Joins(`JOIN (?) t2 
                ON notification_template.templateid = t2.templateid 
               AND notification_template.version = t2.version`, sub)

		return fetch(baseQuery, &configs)
	}

	// Case 3: tenantId + ids (+ templateId optional)
	if search.TemplateID != "" {
		baseQuery = baseQuery.Where("notification_template.templateid = ?", search.TemplateID)
	}

	return fetch(baseQuery, &configs)
}

func fetch(tx *gorm.DB, configs *[]models.TemplateDB) ([]models.TemplateDB, error) {
	err := tx.Find(configs).Error
	return *configs, err
}

func (r *TemplateRepository) Delete(templateID, tenantID string, version int) error {
	return r.db.Where("tenantid = ? AND templateid = ? AND version = ?", tenantID, templateID, version).Delete(&models.TemplateDB{}).Error
}
