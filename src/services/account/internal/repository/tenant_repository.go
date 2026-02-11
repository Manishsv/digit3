package repository

import (
	"account/internal/models"
	"context"
	"database/sql"
)

type TenantRepository struct {
	DB *sql.DB
}

func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{DB: db}
}

func (r *TenantRepository) CreateTenant(ctx context.Context, t *models.Tenant) error {
	query := `INSERT INTO tenant_v1 (id, code, name, email, additionalAttributes, isActive, tenantId, createdBy, lastModifiedBy, createdTime, lastModifiedTime) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`
	_, err := r.DB.ExecContext(ctx, query, t.ID, t.Code, t.Name, t.Email, t.AdditionalAttributes, t.IsActive, t.TenantID, t.CreatedBy, t.LastModifiedBy, t.CreatedTime, t.LastModifiedTime)
	return err
}

func (r *TenantRepository) GetTenantByID(ctx context.Context, id string) (*models.Tenant, error) {
	query := `SELECT id, code, name, email, additionalAttributes, isActive, tenantId, createdBy, lastModifiedBy, createdTime, lastModifiedTime FROM tenant_v1 WHERE id = $1`
	row := r.DB.QueryRowContext(ctx, query, id)
	t := &models.Tenant{}
	err := row.Scan(&t.ID, &t.Code, &t.Name, &t.Email, &t.AdditionalAttributes, &t.IsActive, &t.TenantID, &t.CreatedBy, &t.LastModifiedBy, &t.CreatedTime, &t.LastModifiedTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *TenantRepository) ListTenants(ctx context.Context) ([]*models.Tenant, error) {
	query := `SELECT id, code, name, email, additionalAttributes, isActive, tenantId, createdBy, lastModifiedBy, createdTime, lastModifiedTime FROM tenant_v1`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tenants []*models.Tenant
	for rows.Next() {
		t := &models.Tenant{}
		err := rows.Scan(&t.ID, &t.Code, &t.Name, &t.Email, &t.AdditionalAttributes, &t.IsActive, &t.TenantID, &t.CreatedBy, &t.LastModifiedBy, &t.CreatedTime, &t.LastModifiedTime)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func (r *TenantRepository) UpdateTenant(ctx context.Context, t *models.Tenant) error {
	query := `UPDATE tenant_v1 SET code=$2, name=$3, email=$4, additionalAttributes=$5, isActive=$6, tenantId=$7, createdBy=$8, lastModifiedBy=$9, createdTime=$10, lastModifiedTime=$11 WHERE id=$1`
	_, err := r.DB.ExecContext(ctx, query, t.ID, t.Code, t.Name, t.Email, t.AdditionalAttributes, t.IsActive, t.TenantID, t.CreatedBy, t.LastModifiedBy, t.CreatedTime, t.LastModifiedTime)
	return err
}

func (r *TenantRepository) DeleteTenant(ctx context.Context, id string) error {
	query := `DELETE FROM tenant_v1 WHERE id = $1`
	_, err := r.DB.ExecContext(ctx, query, id)
	return err
}
