package postgres

import (
	"context"
	"fmt"
	"regexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"localization/internal/core/domain"
	"localization/internal/core/ports"
)

// MessageRepositoryImpl is an implementation of the MessageRepository using GORM
type MessageRepositoryImpl struct {
	db *gorm.DB
}

// NewMessageRepository creates a new instance of MessageRepositoryImpl
func NewMessageRepository(db *gorm.DB) ports.MessageRepository {
	return &MessageRepositoryImpl{
		db: db,
	}
}

// isValidUUID checks if a string is a valid UUID format
func isValidUUID(u string) bool {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidRegex.MatchString(u)
}

// SaveMessages persists messages to the database using GORM batch operations for better performance
func (r *MessageRepositoryImpl) SaveMessages(ctx context.Context, messages []domain.Message) error {
	if len(messages) == 0 {
		return nil
	}

	// Use GORM's transaction with ON CONFLICT handling
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Use GORM's ON CONFLICT clause for efficient conflict handling
		// This avoids individual SELECT queries and uses database-level conflict detection
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "locale"}, {Name: "module"}, {Name: "code"}},
			DoNothing: true, // Skip conflicting records silently
		}).CreateInBatches(&messages, 100) // Process in batches of 100

		if result.Error != nil {
			return result.Error
		}

		// If no rows were affected, it means all messages conflicted
		if result.RowsAffected == 0 {
			// Find which message caused the conflict (only when needed)
			for _, msg := range messages {
				var count int64
				err := tx.Model(&domain.Message{}).
					Where("tenant_id = ? AND locale = ? AND module = ? AND code = ?",
						msg.TenantID, msg.Locale, msg.Module, msg.Code).
					Count(&count).Error
				if err != nil {
					return err
				}
				if count > 0 {
					return fmt.Errorf("message with code '%s' already exists for module '%s' and locale '%s'. Duplicate codes are not allowed",
						msg.Code, msg.Module, msg.Locale)
				}
			}
		}

		return nil
	})
}

// UpdateMessages updates existing messages by their UUIDs using GORM with optimized batch operations
func (r *MessageRepositoryImpl) UpdateMessages(ctx context.Context, tenantID string, messages []domain.Message) ([]domain.Message, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	var updatedMessages []domain.Message

	return updatedMessages, r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Validate UUID formats
		uuids := make([]string, len(messages))
		for i, msg := range messages {
			uuids[i] = msg.UUID
			if !isValidUUID(msg.UUID) {
				return fmt.Errorf("invalid UUID format: '%s'. UUID must be in format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", msg.UUID)
			}
		}

		// Batch validation: Check if all UUIDs exist for the tenant in one query
		var count int64
		err := tx.Model(&domain.Message{}).
			Where("tenant_id = ? AND uuid IN ?", tenantID, uuids).
			Count(&count).Error
		if err != nil {
			return fmt.Errorf("database error while validating UUIDs: %w", err)
		}

		if count != int64(len(uuids)) {
			missingCount := int64(len(uuids)) - count
			return fmt.Errorf("update failed: %d UUID(s) not found in database. All UUIDs must exist to perform atomic update", missingCount)
		}

		// Fetch existing records to return complete data
		var existingMessages []domain.Message
		err = tx.Where("tenant_id = ? AND uuid IN ?", tenantID, uuids).Find(&existingMessages).Error
		if err != nil {
			return fmt.Errorf("error fetching existing records: %w", err)
		}

		// Create a map for quick lookup
		existingMap := make(map[string]domain.Message)
		for _, existing := range existingMessages {
			existingMap[existing.UUID] = existing
		}

		// Update messages and build result
		updatedMessages = make([]domain.Message, 0, len(messages))
		for _, msg := range messages {
			// Update the specific message
			err := tx.Model(&domain.Message{}).
				Where("tenant_id = ? AND uuid = ?", tenantID, msg.UUID).
				Updates(map[string]interface{}{
					"message":            msg.Message,
					"last_modified_by":   msg.LastModifiedBy,
					"last_modified_date": msg.LastModifiedDate,
				}).Error

			if err != nil {
				return fmt.Errorf("unexpected error updating UUID %s: %w", msg.UUID, err)
			}

			// Build the response using existing data + updated fields
			if existing, ok := existingMap[msg.UUID]; ok {
				updatedMsg := existing
				updatedMsg.Message = msg.Message
				updatedMsg.LastModifiedBy = msg.LastModifiedBy
				updatedMsg.LastModifiedDate = msg.LastModifiedDate
				updatedMessages = append(updatedMessages, updatedMsg)
			}
		}

		return nil
	})
}

// DeleteMessages deletes messages by their UUIDs for a given tenant using GORM batch operations
func (r *MessageRepositoryImpl) DeleteMessages(ctx context.Context, tenantID string, uuids []string) error {
	if len(uuids) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Validate UUID formats
		for _, uuid := range uuids {
			if !isValidUUID(uuid) {
				return fmt.Errorf("invalid UUID format: '%s'. UUID must be in format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", uuid)
			}
		}

		// Batch validation: Check if all UUIDs exist for the tenant
		var count int64
		err := tx.Model(&domain.Message{}).
			Where("tenant_id = ? AND uuid IN ?", tenantID, uuids).
			Count(&count).Error
		if err != nil {
			return fmt.Errorf("database error while validating UUIDs: %w", err)
		}

		if count != int64(len(uuids)) {
			missingCount := int64(len(uuids)) - count
			return fmt.Errorf("delete failed: %d UUID(s) not found in database. All UUIDs must exist to perform atomic deletion", missingCount)
		}

		// Batch delete operation
		result := tx.Where("tenant_id = ? AND uuid IN ?", tenantID, uuids).Delete(&domain.Message{})
		return result.Error
	})
}

// FindMessages finds messages based on the search criteria, with pagination and sorting using GORM
func (r *MessageRepositoryImpl) FindMessages(ctx context.Context, tenantID, module, locale string, limit, offset int) ([]domain.Message, error) {
	var messages []domain.Message

	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)

	// Add module filter if provided
	if module != "" {
		query = query.Where("module = ?", module)
	}

	// Add locale filter if provided
	if locale != "" {
		query = query.Where("locale = ?", locale)
	}

	// Add ordering
	query = query.Order("code ASC")

	// Add pagination if limit is specified
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err := query.Find(&messages).Error
	return messages, err
}

// FindMessagesByCode finds messages with specific codes using GORM with optimized IN queries
func (r *MessageRepositoryImpl) FindMessagesByCode(ctx context.Context, tenantID, locale string, codes []string) ([]domain.Message, error) {
	if len(codes) == 0 {
		return []domain.Message{}, nil
	}

	var messages []domain.Message
	query := r.db.WithContext(ctx).Where("tenant_id = ? AND code IN ?", tenantID, codes)

	// Add locale filter if provided
	if locale != "" {
		query = query.Where("locale = ?", locale)
	}

	err := query.Find(&messages).Error
	return messages, err
}

// FindAllMessages fetches all messages from the database using GORM with memory-efficient batch loading
func (r *MessageRepositoryImpl) FindAllMessages(ctx context.Context) ([]domain.Message, error) {
	var allMessages []domain.Message

	// Use GORM's batch processing for memory-efficient loading of large datasets
	err := r.db.WithContext(ctx).FindInBatches(&allMessages, 1000, func(tx *gorm.DB, batch int) error {
		// Each batch is automatically appended to allMessages
		// We can add processing logic here if needed in the future
		return nil
	}).Error

	return allMessages, err
}
