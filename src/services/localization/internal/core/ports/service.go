package ports

import (
	"context"

	"localization/internal/core/domain"
)

// MessageService defines the business operations for localization messages
type MessageService interface {
	// UpsertMessages creates or updates localization messages
	UpsertMessages(ctx context.Context, tenantID string, userID string, messages []domain.Message) ([]domain.Message, error)

	// SearchMessages retrieves messages based on search criteria
	SearchMessages(ctx context.Context, tenantID, module, locale string, limit, offset int) ([]domain.Message, error)

	// SearchMessagesByCodes retrieves messages for specific codes
	SearchMessagesByCodes(ctx context.Context, tenantID, locale string, codes []string) ([]domain.Message, error)

	// CreateMessages creates new localization messages
	CreateMessages(ctx context.Context, tenantID string, userID string, messages []domain.Message) ([]domain.Message, error)

	// UpdateMessages updates existing messages by their UUIDs
	UpdateMessages(ctx context.Context, tenantID string, userID string, messages []domain.Message) ([]domain.Message, error)

	// DeleteMessages deletes messages by their UUIDs
	DeleteMessages(ctx context.Context, tenantID string, uuids []string) error

	// BustCache clears the cache based on tenant, and optionally module and locale
	BustCache(ctx context.Context, tenantID, module, locale string) error

	// LoadAllMessages loads all messages from the repository and builds the tenant-to-code-to-locales map
	LoadAllMessages(ctx context.Context) error

	// FindMissingMessages finds the missing messages for a given tenant and an optional module.
	FindMissingMessages(ctx context.Context, tenantID string, module string) (map[string]map[string][]string, error)
}
