package enrichment

import (
	"account/internal/models"
	"account/internal/util"
	"time"

	"github.com/google/uuid"
)

func EnrichTenant(tenant *models.Tenant, clientID string) {
	now := time.Now().Unix()
	if tenant.ID == "" {
		tenant.ID = uuid.New().String()
	}
	if tenant.Code == "" && tenant.Name != "" {
		tenant.Code = util.GenerateCodeFromName(tenant.Name)
	}
	tenant.CreatedBy = clientID
	tenant.LastModifiedBy = clientID
	tenant.CreatedTime = now
	tenant.LastModifiedTime = now
}

func EnrichTenantUpdate(tenant *models.Tenant, clientID string) {
	tenant.LastModifiedBy = clientID
	tenant.LastModifiedTime = time.Now().Unix()
}
