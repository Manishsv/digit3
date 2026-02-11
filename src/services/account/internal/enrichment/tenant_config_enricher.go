package enrichment

import (
	"account/internal/models"
	"account/internal/util"
	"time"

	"github.com/google/uuid"
)

func EnrichTenantConfig(config *models.TenantConfig, clientID string) {
	now := time.Now().Unix()
	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	if config.Code == "" && config.Name != "" {
		config.Code = util.GenerateCodeFromName(config.Name)
	}
	config.AuditDetails.CreatedBy = clientID
	config.AuditDetails.LastModifiedBy = clientID
	config.AuditDetails.CreatedTime = now
	config.AuditDetails.LastModifiedTime = now

	for i := range config.Documents {
		doc := &config.Documents[i]
		if doc.ID == "" {
			doc.ID = uuid.New().String()
		}
		doc.TenantConfigID = config.ID
		doc.AuditDetails.CreatedBy = clientID
		doc.AuditDetails.LastModifiedBy = clientID
		doc.AuditDetails.CreatedTime = now
		doc.AuditDetails.LastModifiedTime = now
	}
}

func EnrichTenantConfigUpdate(config *models.TenantConfig, clientID string) {
	config.AuditDetails.LastModifiedBy = clientID
	config.AuditDetails.LastModifiedTime = time.Now().Unix()
	for i := range config.Documents {
		doc := &config.Documents[i]
		doc.AuditDetails.LastModifiedBy = clientID
		doc.AuditDetails.LastModifiedTime = time.Now().Unix()
	}
}

func EnrichDocument(doc *models.Document, clientID string) {
	now := time.Now().Unix()
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	doc.AuditDetails.CreatedBy = clientID
	doc.AuditDetails.LastModifiedBy = clientID
	doc.AuditDetails.CreatedTime = now
	doc.AuditDetails.LastModifiedTime = now
}

func EnrichDocumentUpdate(doc *models.Document, clientID string) {
	doc.AuditDetails.LastModifiedBy = clientID
	doc.AuditDetails.LastModifiedTime = time.Now().Unix()
}
