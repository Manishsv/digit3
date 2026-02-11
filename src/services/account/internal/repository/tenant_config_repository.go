package repository

import (
	"account/internal/models"
	"context"
	"database/sql"

	"github.com/lib/pq"
)

type TenantConfigRepository struct {
	DB           *sql.DB
	DocumentRepo *DocumentRepository
}

func NewTenantConfigRepository(db *sql.DB) *TenantConfigRepository {
	return &TenantConfigRepository{
		DB:           db,
		DocumentRepo: NewDocumentRepository(db),
	}
}

func (r *TenantConfigRepository) CreateTenantConfig(ctx context.Context, t *models.TenantConfig) error {
	query := `INSERT INTO tenant_config_v1 (id, code, defaultLoginType, otpLength, name, enableUserBasedLogin, additionalAttributes, isActive, languages, createdBy, lastModifiedBy, createdTime, lastModifiedTime) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`
	_, err := r.DB.ExecContext(ctx, query, t.ID, t.Code, t.DefaultLoginType, t.OtpLength, t.Name, t.EnableUserBasedLogin, t.AdditionalAttributes, t.IsActive, pq.Array(t.Languages), t.AuditDetails.CreatedBy, t.AuditDetails.LastModifiedBy, t.AuditDetails.CreatedTime, t.AuditDetails.LastModifiedTime)
	if err != nil {
		return err
	}
	// Insert documents if any
	if len(t.Documents) > 0 {
		docQuery := `INSERT INTO tenant_documents_v1 (id, tenantConfigId, tenantId, type, fileStoreId, url, isActive, createdBy, lastModifiedBy, createdTime, lastModifiedTime) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
		for _, doc := range t.Documents {
			_, err := r.DB.ExecContext(ctx, docQuery, doc.ID, doc.TenantConfigID, doc.TenantID, doc.Type, doc.FileStoreID, doc.URL, doc.IsActive, doc.AuditDetails.CreatedBy, doc.AuditDetails.LastModifiedBy, doc.AuditDetails.CreatedTime, doc.AuditDetails.LastModifiedTime)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *TenantConfigRepository) GetTenantConfigByID(ctx context.Context, id string) (*models.TenantConfig, error) {
	query := `SELECT id, code, defaultLoginType, otpLength, name, enableUserBasedLogin, additionalAttributes, isActive, languages, createdBy, lastModifiedBy, createdTime, lastModifiedTime FROM tenant_config_v1 WHERE id = $1`
	row := r.DB.QueryRowContext(ctx, query, id)
	t := &models.TenantConfig{}
	var languages []string
	err := row.Scan(&t.ID, &t.Code, &t.DefaultLoginType, &t.OtpLength, &t.Name, &t.EnableUserBasedLogin, &t.AdditionalAttributes, &t.IsActive, pq.Array(&languages), &t.AuditDetails.CreatedBy, &t.AuditDetails.LastModifiedBy, &t.AuditDetails.CreatedTime, &t.AuditDetails.LastModifiedTime)
	t.Languages = languages
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Fetch documents for this config
	documents, err := r.DocumentRepo.GetDocumentsByTenantConfigID(ctx, t.ID)
	if err != nil {
		return nil, err
	}

	// Convert []*Document to []Document
	var docs []models.Document
	for _, doc := range documents {
		docs = append(docs, *doc)
	}
	t.Documents = docs

	return t, nil
}

func (r *TenantConfigRepository) ListTenantConfigs(ctx context.Context) ([]*models.TenantConfig, error) {
	query := `SELECT id, code, defaultLoginType, otpLength, name, enableUserBasedLogin, additionalAttributes, isActive, languages, createdBy, lastModifiedBy, createdTime, lastModifiedTime FROM tenant_config_v1`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []*models.TenantConfig
	for rows.Next() {
		t := &models.TenantConfig{}
		var languages []string
		err := rows.Scan(&t.ID, &t.Code, &t.DefaultLoginType, &t.OtpLength, &t.Name, &t.EnableUserBasedLogin, &t.AdditionalAttributes, &t.IsActive, pq.Array(&languages), &t.AuditDetails.CreatedBy, &t.AuditDetails.LastModifiedBy, &t.AuditDetails.CreatedTime, &t.AuditDetails.LastModifiedTime)
		t.Languages = languages
		if err != nil {
			return nil, err
		}

		// Fetch documents for this config
		documents, err := r.DocumentRepo.GetDocumentsByTenantConfigID(ctx, t.ID)
		if err != nil {
			return nil, err
		}

		// Convert []*Document to []Document
		var docs []models.Document
		for _, doc := range documents {
			docs = append(docs, *doc)
		}
		t.Documents = docs

		configs = append(configs, t)
	}
	return configs, nil
}

func (r *TenantConfigRepository) UpdateTenantConfig(ctx context.Context, t *models.TenantConfig) error {
	// Update all fields except code, createdBy, and createdTime
	query := `UPDATE tenant_config_v1 SET 
		defaultLoginType=$2, 
		otpLength=$3, 
		name=$4, 
		enableUserBasedLogin=$5, 
		additionalAttributes=$6, 
		isActive=$7, 
		languages=$8, 
		lastModifiedBy=$9, 
		lastModifiedTime=$10 
		WHERE id=$1`

	_, err := r.DB.ExecContext(ctx, query,
		t.ID,
		t.DefaultLoginType,
		t.OtpLength,
		t.Name,
		t.EnableUserBasedLogin,
		t.AdditionalAttributes,
		t.IsActive,
		pq.Array(t.Languages),
		t.AuditDetails.LastModifiedBy,
		t.AuditDetails.LastModifiedTime)
	return err
}

func (r *TenantConfigRepository) DeleteTenantConfig(ctx context.Context, id string) error {
	query := `DELETE FROM tenant_config_v1 WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}
