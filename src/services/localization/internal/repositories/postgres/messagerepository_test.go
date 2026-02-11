package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"localization/internal/core/domain"
)

func TestSaveMessages(t *testing.T) {
	// Create mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create repository with mock DB
	repo := NewMessageRepository(db)

	// Test data
	now := time.Now()
	messages := []domain.Message{
		{
			TenantID:         "DEFAULT",
			Module:           "test-module",
			Locale:           "en_IN",
			Code:             "test-code-1",
			Message:          "Test Message 1",
			CreatedBy:        1,
			CreatedDate:      now,
			LastModifiedBy:   1,
			LastModifiedDate: now,
		},
	}

	t.Run("Success", func(t *testing.T) {
		// Expect begin transaction
		mock.ExpectBegin()

		// Expect prepare statement
		mock.ExpectPrepare("INSERT INTO message")

		// Expect execute with params and return ID
		mock.ExpectQuery("INSERT INTO message").
			WithArgs(
				messages[0].TenantID,
				messages[0].Module,
				messages[0].Locale,
				messages[0].Code,
				messages[0].Message,
				messages[0].CreatedBy,
				messages[0].CreatedDate,
				messages[0].LastModifiedBy,
				messages[0].LastModifiedDate,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		// Expect commit
		mock.ExpectCommit()

		// Call the method under test
		err := repo.SaveMessages(context.Background(), messages)

		// Assert no error and expectations met
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Transaction error", func(t *testing.T) {
		// Expect begin transaction with error
		mock.ExpectBegin().WillReturnError(errors.New("tx error"))

		// Call the method under test
		err := repo.SaveMessages(context.Background(), messages)

		// Assert expected error and expectations met
		assert.Error(t, err)
		assert.Equal(t, "tx error", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Prepare statement error", func(t *testing.T) {
		// Expect begin transaction
		mock.ExpectBegin()

		// Expect prepare statement with error
		mock.ExpectPrepare("INSERT INTO message").WillReturnError(errors.New("prepare error"))

		// Expect rollback due to error
		mock.ExpectRollback()

		// Call the method under test
		err := repo.SaveMessages(context.Background(), messages)

		// Assert expected error and expectations met
		assert.Error(t, err)
		assert.Equal(t, "prepare error", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Execute error", func(t *testing.T) {
		// Expect begin transaction
		mock.ExpectBegin()

		// Expect prepare statement
		mock.ExpectPrepare("INSERT INTO message")

		// Expect execute with error
		mock.ExpectQuery("INSERT INTO message").
			WithArgs(
				messages[0].TenantID,
				messages[0].Module,
				messages[0].Locale,
				messages[0].Code,
				messages[0].Message,
				messages[0].CreatedBy,
				messages[0].CreatedDate,
				messages[0].LastModifiedBy,
				messages[0].LastModifiedDate,
			).
			WillReturnError(errors.New("execute error"))

		// Expect rollback due to error
		mock.ExpectRollback()

		// Call the method under test
		err := repo.SaveMessages(context.Background(), messages)

		// Assert expected error and expectations met
		assert.Error(t, err)
		assert.Equal(t, "execute error", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Commit error", func(t *testing.T) {
		// Expect begin transaction
		mock.ExpectBegin()

		// Expect prepare statement
		mock.ExpectPrepare("INSERT INTO message")

		// Expect execute with params and return ID
		mock.ExpectQuery("INSERT INTO message").
			WithArgs(
				messages[0].TenantID,
				messages[0].Module,
				messages[0].Locale,
				messages[0].Code,
				messages[0].Message,
				messages[0].CreatedBy,
				messages[0].CreatedDate,
				messages[0].LastModifiedBy,
				messages[0].LastModifiedDate,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		// Expect commit with error
		mock.ExpectCommit().WillReturnError(errors.New("commit error"))

		// Call the method under test
		err := repo.SaveMessages(context.Background(), messages)

		// Assert expected error and expectations met
		assert.Error(t, err)
		assert.Equal(t, "commit error", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFindMessages(t *testing.T) {
	// Create mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create repository with mock DB
	repo := NewMessageRepository(db)

	// Test data
	now := time.Now()
	tenantID := "DEFAULT"
	module := "test-module"
	locale := "en_IN"
	id := int64(1)
	code := "test-code-1"
	message := "Test Message 1"
	createdBy := int64(1)
	lastModifiedBy := int64(1)

	t.Run("Success with all filters", func(t *testing.T) {
		// Set up rows with test data
		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "module", "locale", "code", "message",
			"created_by", "created_date", "last_modified_by", "last_modified_date",
		}).AddRow(
			id, tenantID, module, locale, code, message,
			createdBy, now, lastModifiedBy, now,
		)

		// Expect query with params
		mock.ExpectQuery("SELECT (.+) FROM message WHERE").
			WithArgs(tenantID, module, locale).
			WillReturnRows(rows)

		// Call the method under test
		messages, err := repo.FindMessages(context.Background(), tenantID, module, locale)

		// Assertions
		assert.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, id, messages[0].ID)
		assert.Equal(t, tenantID, messages[0].TenantID)
		assert.Equal(t, module, messages[0].Module)
		assert.Equal(t, locale, messages[0].Locale)
		assert.Equal(t, code, messages[0].Code)
		assert.Equal(t, message, messages[0].Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Success with only tenant filter", func(t *testing.T) {
		// Set up rows with test data
		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "module", "locale", "code", "message",
			"created_by", "created_date", "last_modified_by", "last_modified_date",
		}).AddRow(
			id, tenantID, module, locale, code, message,
			createdBy, now, lastModifiedBy, now,
		)

		// Expect query with only tenant param
		mock.ExpectQuery("SELECT (.+) FROM message WHERE").
			WithArgs(tenantID).
			WillReturnRows(rows)

		// Call the method under test
		messages, err := repo.FindMessages(context.Background(), tenantID, "", "")

		// Assertions
		assert.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, id, messages[0].ID)
		assert.Equal(t, tenantID, messages[0].TenantID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Success with no results", func(t *testing.T) {
		// Empty result set
		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "module", "locale", "code", "message",
			"created_by", "created_date", "last_modified_by", "last_modified_date",
		})

		// Expect query with params
		mock.ExpectQuery("SELECT (.+) FROM message WHERE").
			WithArgs(tenantID, module, locale).
			WillReturnRows(rows)

		// Call the method under test
		messages, err := repo.FindMessages(context.Background(), tenantID, module, locale)

		// Assertions
		assert.NoError(t, err)
		assert.Empty(t, messages)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query error", func(t *testing.T) {
		// Expect query with error
		mock.ExpectQuery("SELECT (.+) FROM message WHERE").
			WithArgs(tenantID, module, locale).
			WillReturnError(errors.New("query error"))

		// Call the method under test
		messages, err := repo.FindMessages(context.Background(), tenantID, module, locale)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, messages)
		assert.Equal(t, "query error", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Scan error", func(t *testing.T) {
		// Return rows with incorrect data type to cause scan error
		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "module", "locale", "code", "message",
			"created_by", "created_date", "last_modified_by", "last_modified_date",
		}).AddRow(
			"not an int64", tenantID, module, locale, code, message, // ID is string instead of int64
			createdBy, now, lastModifiedBy, now,
		)

		// Expect query with params
		mock.ExpectQuery("SELECT (.+) FROM message WHERE").
			WithArgs(tenantID, module, locale).
			WillReturnRows(rows)

		// Call the method under test
		messages, err := repo.FindMessages(context.Background(), tenantID, module, locale)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, messages)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFindMessagesByCode(t *testing.T) {
	// Create mock DB
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create repository with mock DB
	repo := NewMessageRepository(db)

	// Test data
	now := time.Now()
	tenantID := "DEFAULT"
	module := "test-module"
	locale := "en_IN"
	id := int64(1)
	code := "test-code-1"
	message := "Test Message 1"
	createdBy := int64(1)
	lastModifiedBy := int64(1)
	codes := []string{"test-code-1", "test-code-2"}

	t.Run("Success with codes", func(t *testing.T) {
		// Set up rows with test data
		rows := sqlmock.NewRows([]string{
			"id", "tenant_id", "module", "locale", "code", "message",
			"created_by", "created_date", "last_modified_by", "last_modified_date",
		}).AddRow(
			id, tenantID, module, locale, code, message,
			createdBy, now, lastModifiedBy, now,
		)

		// Use AnyArg() for the codes parameter since we now use dynamic SQL
		mock.ExpectQuery("SELECT (.+) FROM message WHERE").
			WithArgs(tenantID, locale, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(rows)

		// Call the method under test
		messages, err := repo.FindMessagesByCode(context.Background(), tenantID, locale, codes)

		// Assertions
		assert.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, id, messages[0].ID)
		assert.Equal(t, tenantID, messages[0].TenantID)
		assert.Equal(t, module, messages[0].Module)
		assert.Equal(t, locale, messages[0].Locale)
		assert.Equal(t, code, messages[0].Code)
		assert.Equal(t, message, messages[0].Message)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Empty codes", func(t *testing.T) {
		// Call the method under test with empty codes
		messages, err := repo.FindMessagesByCode(context.Background(), tenantID, locale, []string{})

		// Assertions
		assert.NoError(t, err)
		assert.Empty(t, messages)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Query error", func(t *testing.T) {
		// Expect query with error
		mock.ExpectQuery("SELECT (.+) FROM message WHERE").
			WithArgs(tenantID, locale, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("query error"))

		// Call the method under test
		messages, err := repo.FindMessagesByCode(context.Background(), tenantID, locale, codes)

		// Assertions
		assert.Error(t, err)
		assert.Nil(t, messages)
		assert.Equal(t, "query error", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
