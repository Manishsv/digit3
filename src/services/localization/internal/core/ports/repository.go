package ports

import (
	"context"

	"localization/internal/core/domain"
)

// MessageRepository defines the operations available for message storage
type MessageRepository interface {
	// SaveMessages persists messages to the database
	SaveMessages(ctx context.Context, messages []domain.Message) error

	// FindMessages finds messages based on the search criteria
	FindMessages(ctx context.Context, tenantID, module, locale string, limit, offset int) ([]domain.Message, error)

	// FindMessagesByCode finds messages with specific codes
	FindMessagesByCode(ctx context.Context, tenantID, locale string, codes []string) ([]domain.Message, error)

	// UpdateMessages updates existing messages by their UUIDs and returns the updated messages
	UpdateMessages(ctx context.Context, tenantID string, messages []domain.Message) ([]domain.Message, error)

	// DeleteMessages deletes messages by their UUIDs for a given tenant
	DeleteMessages(ctx context.Context, tenantID string, uuids []string) error

	// FindAllMessages fetches all messages from the database
	FindAllMessages(ctx context.Context) ([]domain.Message, error)
}

// HealthRepository defines the interface for health checks
type HealthRepository interface {
	// CheckHealth checks the health of the system
	CheckHealth() error
}
