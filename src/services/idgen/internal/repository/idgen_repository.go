package repository

import (
	"crypto/sha1"
	"fmt"
	"idgen/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IDGenRepository struct {
	db *gorm.DB
}

func NewIDGenRepository(db *gorm.DB) *IDGenRepository {
	return &IDGenRepository{db: db}
}

func (r *IDGenRepository) CreateTemplate(template *models.IDGenTemplateDB) error {
	return r.db.Create(template).Error
}

func (r *IDGenRepository) GetLatestTemplate(templateCode, tenantID string) (*models.IDGenTemplateDB, error) {
	var config models.IDGenTemplateDB
	err := r.db.
		Where("tenantid = ? AND templatecode = ?", tenantID, templateCode).
		Order("version DESC").
		First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *IDGenRepository) SearchTemplates(search *models.IDGenTemplateSearch) ([]models.IDGenTemplateDB, error) {
	var templates []models.IDGenTemplateDB

	// Base query with tenant filter (mandatory)
	baseQuery := r.db.Model(&models.IDGenTemplateDB{}).
		Where("idgen_templates.tenantid = ?", search.TenantID)

		// IDs filter (applies to all cases)
	if len(search.IDs) > 0 {
		baseQuery = baseQuery.Where("idgen_templates.id IN ?", search.IDs)
	}

	// Version explicitly provided
	if search.VersionInt > 0 {
		if search.TemplateCode != "" {
			baseQuery = baseQuery.Where("idgen_templates.templatecode = ?", search.TemplateCode)
		}
		baseQuery = baseQuery.Where("idgen_templates.version = ?", search.VersionInt)
		return fetch(baseQuery, &templates)
	}

	// Case 1: tenantId + templateId (no version, no IDs)
	if search.TemplateCode != "" && len(search.IDs) == 0 {
		sub := r.db.
			Table("idgen_templates").
			Select("templatecode, MAX(version) as version").
			Where("tenantid = ?", search.TenantID).
			Group("templatecode")

		baseQuery = baseQuery.
			Joins(`JOIN (?) t2 
                ON idgen_templates.templatecode = t2.templatecode 
               AND idgen_templates.version = t2.version`, sub).
			Where("idgen_templates.templatecode = ?", search.TemplateCode)

		return fetch(baseQuery, &templates)
	}

	// Case 2: tenantId only (latest version of each template)
	if search.TemplateCode == "" && len(search.IDs) == 0 {
		sub := r.db.
			Table("idgen_templates").
			Select("templatecode, MAX(version) as version").
			Where("tenantid = ?", search.TenantID).
			Group("templatecode")

		baseQuery = baseQuery.
			Joins(`JOIN (?) t2 
                ON idgen_templates.templatecode = t2.templatecode 
               AND idgen_templates.version = t2.version`, sub)

		return fetch(baseQuery, &templates)
	}

	// Case 3: tenantId + ids (+ templateId optional)
	if search.TemplateCode != "" {
		baseQuery = baseQuery.Where("idgen_templates.templatecode = ?", search.TemplateCode)
	}

	return fetch(baseQuery, &templates)
}

func fetch(tx *gorm.DB, templates *[]models.IDGenTemplateDB) ([]models.IDGenTemplateDB, error) {
	err := tx.Find(templates).Error
	return *templates, err
}

func (r *IDGenRepository) DeleteTemplate(tenantID, templateCode string, version int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if tenantID == "" || templateCode == "" || version <= 0 {
			return fmt.Errorf("tenantID, templateCode, and valid version are required")
		}

		// --- Get latest version ---
		var latestVersion int
		if err := tx.Model(&models.IDGenTemplateDB{}).
			Where("tenantid = ? AND templatecode = ?", tenantID, templateCode).
			Select("COALESCE(MAX(version), 0)").
			Scan(&latestVersion).Error; err != nil {
			return err
		}
		if latestVersion == 0 {
			return gorm.ErrRecordNotFound
		}

		// --- Count total versions ---
		var versionCount int64
		if err := tx.Model(&models.IDGenTemplateDB{}).
			Where("tenantid = ? AND templatecode = ?", tenantID, templateCode).
			Count(&versionCount).Error; err != nil {
			return err
		}

		// --- CASE 1: version < latest → delete only that row ---
		if version < latestVersion {
			if err := tx.Where("tenantid = ? AND templatecode = ? AND version = ?",
				tenantID, templateCode, version).
				Delete(&models.IDGenTemplateDB{}).Error; err != nil {
				return err
			}
			return nil
		}

		// --- CASE 2: version == latest ---
		if version == latestVersion {
			if versionCount > 1 {
				// delete only this version; keep sequence + lookup + resets
				if err := tx.Where("tenantid = ? AND templatecode = ? AND version = ?",
					tenantID, templateCode, version).
					Delete(&models.IDGenTemplateDB{}).Error; err != nil {
					return err
				}
				return nil
			}

			// CASE 3: only one version exists → full cleanup
			// 1. Delete template
			if err := tx.Where("tenantid = ? AND templatecode = ?", tenantID, templateCode).
				Delete(&models.IDGenTemplateDB{}).Error; err != nil {
				return err
			}

			// 2. Delete sequence resets
			if err := tx.Where("tenantid = ? AND templatecode = ?", tenantID, templateCode).
				Delete(&models.IDGenSequenceReset{}).Error; err != nil {
				return err
			}

			// 3. Drop the sequence if it exists
			var lookup models.IDGenSequenceLookup
			if err := tx.Where("tenantid = ? AND templatecode = ?", tenantID, templateCode).
				First(&lookup).Error; err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			if lookup.SeqName != "" {
				dropSQL := fmt.Sprintf("DROP SEQUENCE IF EXISTS %s CASCADE", lookup.SeqName)
				if err := tx.Exec(dropSQL).Error; err != nil {
					return err
				}
			}

			// 4. Delete lookup row
			if err := tx.Where("tenantid = ? AND templatecode = ?", tenantID, templateCode).
				Delete(&models.IDGenSequenceLookup{}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *IDGenRepository) GetByTenantIDTemplateCodeAndVersion(tenantID, templateCode string, version int) (*models.IDGenTemplateDB, error) {
	var template models.IDGenTemplateDB
	err := r.db.Where("tenantid = ? AND templatecode = ? AND version = ?", tenantID, templateCode, version).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *IDGenRepository) CreateSequence(tenantID, templateCode string, start int) error {
	seqName := generateSequenceName(tenantID, templateCode)

	return r.db.Transaction(func(tx *gorm.DB) error {
		// Create sequence
		createSequenceSQL := fmt.Sprintf(`
			CREATE SEQUENCE IF NOT EXISTS %s
			START WITH %d
			INCREMENT BY 1
			MINVALUE 1
			CACHE 1;
		`, seqName, start)
		if err := tx.Exec(createSequenceSQL).Error; err != nil {
			return err
		}

		// Insert mapping using GORM
		lookup := &models.IDGenSequenceLookup{
			ID:           uuid.New(),
			SeqName:      seqName,
			TenantID:     tenantID,
			TemplateCode: templateCode,
		}
		return tx.Create(lookup).Error
	})
}

func (r *IDGenRepository) NextSequenceValue(tenantID, templateCode string) (int64, error) {
	seqName := generateSequenceName(tenantID, templateCode)
	var value int64
	query := fmt.Sprintf("SELECT nextval('%s')", seqName)
	if err := r.db.Raw(query).Scan(&value).Error; err != nil {
		return 0, err
	}
	return value, nil
}

func (r *IDGenRepository) EnsureScopeReset(tenantID, templateCode, scopeKey string, start int) error {
	lockID := generateLockID(tenantID, templateCode, scopeKey) // int64 hash

	return r.db.Transaction(func(tx *gorm.DB) error {
		// Acquire advisory lock (transaction-scoped)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", lockID).Error; err != nil {
			return err
		}

		// Check if reset already exists
		var count int64
		if err := tx.Model(&models.IDGenSequenceReset{}).
			Where("tenantid = ? AND templatecode = ? AND scopekey = ?", tenantID, templateCode, scopeKey).
			Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			// Already reset, nothing to do
			return nil
		}

		// Reset the underlying Postgres sequence
		seqName := generateSequenceName(tenantID, templateCode)
		if err := tx.Exec(fmt.Sprintf("ALTER SEQUENCE %s RESTART WITH %d", seqName, start)).Error; err != nil {
			return err
		}

		// Track the reset in idgen_sequence_resets table
		reset := &models.IDGenSequenceReset{
			ID:           uuid.New(),
			TenantID:     tenantID,
			TemplateCode: templateCode,
			ScopeKey:     scopeKey,
			LastValue:    0,
		}
		if err := tx.Create(&reset).Error; err != nil {
			return err
		}

		return nil
	})
}

func generateLockID(tenantID, templateCode, scopeKey string) int64 {
	// Simple hash to int64 (advisory lock requires int64)
	h := sha1.Sum([]byte(fmt.Sprintf("%s:%s:%s", tenantID, templateCode, scopeKey)))
	// Take first 8 bytes
	return int64(h[0])<<56 | int64(h[1])<<48 | int64(h[2])<<40 | int64(h[3])<<32 |
		int64(h[4])<<24 | int64(h[5])<<16 | int64(h[6])<<8 | int64(h[7])
}

// generateSequenceName creates a deterministic, unique, Postgres-safe sequence name.
// Format: seq_v1_<SHA1(tenantID + ":" + templateCode)>
// Example: seq_v1_2c75e5f85ac9dc8085af8b8a931bda9ef0a6ed60
func generateSequenceName(tenantID, templateCode string) string {
	const prefix = "seq_v1_" // allows future format changes
	base := fmt.Sprintf("%s:%s", tenantID, templateCode)
	hash := sha1.Sum([]byte(base))
	return fmt.Sprintf("%s%x", prefix, hash)
}
