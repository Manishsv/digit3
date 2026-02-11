package repository

import (
	"account/internal/models"
	"context"
	"database/sql"
)

type DocumentRepository struct {
	DB *sql.DB
}

func NewDocumentRepository(db *sql.DB) *DocumentRepository {
	return &DocumentRepository{DB: db}
}

func (r *DocumentRepository) CreateDocument(ctx context.Context, d *models.Document) error {
	query := `INSERT INTO tenant_documents_v1 (id, tenantId, tenantConfigId, type, fileStoreId, url, isActive, createdBy, lastModifiedBy, createdTime, lastModifiedTime) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
	_, err := r.DB.ExecContext(ctx, query, d.ID, d.TenantID, d.TenantConfigID, d.Type, d.FileStoreID, d.URL, d.IsActive, d.AuditDetails.CreatedBy, d.AuditDetails.LastModifiedBy, d.AuditDetails.CreatedTime, d.AuditDetails.LastModifiedTime)
	return err
}

func (r *DocumentRepository) GetDocumentByID(ctx context.Context, id string) (*models.Document, error) {
	query := `SELECT id, tenantId, tenantConfigId, type, fileStoreId, url, isActive, createdBy, lastModifiedBy, createdTime, lastModifiedTime FROM tenant_documents_v1 WHERE id = $1`
	row := r.DB.QueryRowContext(ctx, query, id)
	d := &models.Document{}
	err := row.Scan(&d.ID, &d.TenantID, &d.TenantConfigID, &d.Type, &d.FileStoreID, &d.URL, &d.IsActive, &d.AuditDetails.CreatedBy, &d.AuditDetails.LastModifiedBy, &d.AuditDetails.CreatedTime, &d.AuditDetails.LastModifiedTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (r *DocumentRepository) ListDocuments(ctx context.Context) ([]*models.Document, error) {
	query := `SELECT id, tenantId, tenantConfigId, type, fileStoreId, url, isActive, createdBy, lastModifiedBy, createdTime, lastModifiedTime FROM tenant_documents_v1`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []*models.Document
	for rows.Next() {
		d := &models.Document{}
		err := rows.Scan(&d.ID, &d.TenantID, &d.TenantConfigID, &d.Type, &d.FileStoreID, &d.URL, &d.IsActive, &d.AuditDetails.CreatedBy, &d.AuditDetails.LastModifiedBy, &d.AuditDetails.CreatedTime, &d.AuditDetails.LastModifiedTime)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func (r *DocumentRepository) UpdateDocument(ctx context.Context, d *models.Document) error {
	query := `UPDATE tenant_documents_v1 SET tenantId=$2, tenantConfigId=$3, type=$4, fileStoreId=$5, url=$6, isActive=$7, createdBy=$8, lastModifiedBy=$9, createdTime=$10, lastModifiedTime=$11 WHERE id=$1`
	_, err := r.DB.ExecContext(ctx, query, d.ID, d.TenantID, d.TenantConfigID, d.Type, d.FileStoreID, d.URL, d.IsActive, d.AuditDetails.CreatedBy, d.AuditDetails.LastModifiedBy, d.AuditDetails.CreatedTime, d.AuditDetails.LastModifiedTime)
	return err
}

func (r *DocumentRepository) DeleteDocument(ctx context.Context, id string) error {
	query := `DELETE FROM tenant_documents_v1 WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}

func (r *DocumentRepository) GetDocumentsByTenantConfigID(ctx context.Context, tenantConfigID string) ([]*models.Document, error) {
	query := `SELECT id, tenantId, tenantConfigId, type, fileStoreId, url, isActive, createdBy, lastModifiedBy, createdTime, lastModifiedTime FROM tenant_documents_v1 WHERE tenantConfigId = $1 AND isActive = true`
	rows, err := r.DB.QueryContext(ctx, query, tenantConfigID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []*models.Document
	for rows.Next() {
		d := &models.Document{}
		err := rows.Scan(&d.ID, &d.TenantID, &d.TenantConfigID, &d.Type, &d.FileStoreID, &d.URL, &d.IsActive, &d.AuditDetails.CreatedBy, &d.AuditDetails.LastModifiedBy, &d.AuditDetails.CreatedTime, &d.AuditDetails.LastModifiedTime)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}
