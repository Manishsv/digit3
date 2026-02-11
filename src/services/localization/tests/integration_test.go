package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"localization/internal/common/models"
	"localization/internal/core/domain"
	"localization/internal/core/services"
	"localization/internal/handlers"
	"localization/internal/platform/cache"
	"localization/internal/repositories/postgres"
	"localization/pkg/dtos"
)

// setupIntegrationTest creates a test environment with in-memory Redis
func setupIntegrationTest(t *testing.T) (*gin.Engine, *miniredis.Miniredis) {
	// Use Gin test mode
	gin.SetMode(gin.TestMode)

	// Set up mini Redis for testing
	miniRedis, err := miniredis.Run()
	require.NoError(t, err)

	// Create Redis client connected to mini Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})

	// Use an in-memory SQLite database for testing
	db, err := setupInMemoryDB()
	require.NoError(t, err)

	// Create the repository, cache, and service
	messageRepo := postgres.NewMessageRepository(db)
	messageCache := cache.NewRedisCache(redisClient)
	messageService := services.NewMessageService(messageRepo, messageCache)

	// Create the handler and router
	handler := handlers.NewMessageHandler(messageService)
	router := gin.Default()
	apiGroup := router.Group("")
	handler.RegisterRoutes(apiGroup)

	return router, miniRedis
}

// TestIntegrationFlow tests the complete flow of upsert and search
func TestIntegrationFlow(t *testing.T) {
	// Skip when running in CI
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test environment
	router, miniRedis := setupIntegrationTest(t)
	defer miniRedis.Close()

	// Test data
	tenantID := "DEFAULT"
	module := "test-module"
	locale := "en_IN"
	messages := []domain.Message{
		{
			TenantID: tenantID,
			Module:   module,
			Locale:   locale,
			Code:     "test-code-1",
			Message:  "Test Message 1",
		},
		{
			TenantID: tenantID,
			Module:   module,
			Locale:   locale,
			Code:     "test-code-2",
			Message:  "Test Message 2",
		},
	}
	requestInfo := models.RequestInfo{
		APIId:     "test",
		Ver:       "1.0",
		Action:    "create",
		Did:       "1",
		Key:       "test-key",
		MsgId:     "test-msg-id",
		AuthToken: "test-token",
		UserInfo: &models.UserInfo{
			ID: 1,
		},
	}

	// Step 1: Upsert messages
	t.Run("Upsert messages", func(t *testing.T) {
		// Create request body
		reqBody := dtos.UpsertMessagesRequest{
			RequestInfo: requestInfo,
			TenantId:    tenantID,
			Messages:    messages,
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Create request
		req, err := http.NewRequest("POST", "/localization/messages/v1/_upsert", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response
		var response dtos.UpsertMessagesResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Messages, 2)
		assert.Equal(t, messages[0].Code, response.Messages[0].Code)
		assert.Equal(t, messages[0].Message, response.Messages[0].Message)
		assert.Equal(t, messages[1].Code, response.Messages[1].Code)
		assert.Equal(t, messages[1].Message, response.Messages[1].Message)
	})

	// Step 2: Search messages using JSON body
	t.Run("Search messages with JSON body", func(t *testing.T) {
		// Create request body
		reqBody := dtos.SearchMessagesRequest{
			RequestInfo: requestInfo,
			TenantId:    tenantID,
			Module:      module,
			Locale:      locale,
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Create request
		req, err := http.NewRequest("POST", "/localization/messages/v1/_search", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response
		var response dtos.SearchMessagesResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Messages, 2)
		assert.Contains(t, []string{response.Messages[0].Code, response.Messages[1].Code}, "test-code-1")
		assert.Contains(t, []string{response.Messages[0].Code, response.Messages[1].Code}, "test-code-2")
	})

	// Step 3: Search messages using query parameters
	t.Run("Search messages with query parameters", func(t *testing.T) {
		// Create request
		reqURL := fmt.Sprintf("/localization/messages/v1/_search?tenantId=%s&module=%s&locale=%s", tenantID, module, locale)
		req, err := http.NewRequest("POST", reqURL, nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response
		var response dtos.SearchMessagesResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Messages, 2)
	})

	// Step 4: Search messages by specific codes
	t.Run("Search messages by codes", func(t *testing.T) {
		// Create request body
		reqBody := dtos.SearchMessagesRequest{
			RequestInfo: requestInfo,
			TenantId:    tenantID,
			Locale:      locale,
			Codes:       []string{"test-code-1"},
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Create request
		req, err := http.NewRequest("POST", "/localization/messages/v1/_search", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response
		var response dtos.SearchMessagesResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Messages, 1)
		assert.Equal(t, "test-code-1", response.Messages[0].Code)
	})

	// Step 5: Create new messages
	t.Run("Create messages", func(t *testing.T) {
		// Define new messages
		newMessages := []domain.Message{
			{
				TenantID: tenantID,
				Module:   module,
				Locale:   locale,
				Code:     "test-code-3",
				Message:  "Test Message 3",
			},
			{
				TenantID: tenantID,
				Module:   module,
				Locale:   locale,
				Code:     "test-code-4",
				Message:  "Test Message 4",
			},
		}

		// Create request body
		reqBody := dtos.CreateMessagesRequest{
			RequestInfo: requestInfo,
			TenantId:    tenantID,
			Messages:    newMessages,
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Create request
		req, err := http.NewRequest("POST", "/localization/messages/v1/_create", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response
		var response dtos.CreateMessagesResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Messages, 2)
		assert.Equal(t, newMessages[0].Code, response.Messages[0].Code)
		assert.Equal(t, newMessages[0].Message, response.Messages[0].Message)
		assert.Equal(t, newMessages[1].Code, response.Messages[1].Code)
		assert.Equal(t, newMessages[1].Message, response.Messages[1].Message)

		// Verify we can search for these new messages
		searchURL := fmt.Sprintf("/localization/v1/messages?tenantId=%s&module=%s&locale=%s&codes=test-code-3,test-code-4", tenantID, module, locale)
		req, err = http.NewRequest("GET", searchURL, nil)
		require.NoError(t, err)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var searchResponse dtos.SearchMessagesResponse
		err = json.Unmarshal(w.Body.Bytes(), &searchResponse)
		require.NoError(t, err)
		assert.Len(t, searchResponse.Messages, 2)
	})

	// Step 6: Update existing messages
	t.Run("Update messages", func(t *testing.T) {
		// Create update messages
		updateMessages := []dtos.UpdateMessage{
			{
				Code:    "test-code-1",
				Message: "Updated Test Message 1",
			},
		}

		// Create request body
		reqBody := dtos.UpdateMessagesRequest{
			RequestInfo: requestInfo,
			TenantId:    tenantID,
			Locale:      locale,
			Module:      module,
			Messages:    updateMessages,
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Create request
		req, err := http.NewRequest("POST", "/localization/messages/v1/_update", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response
		var response dtos.UpdateMessagesResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Messages, 1)
		assert.Equal(t, updateMessages[0].Code, response.Messages[0].Code)
		assert.Equal(t, updateMessages[0].Message, response.Messages[0].Message)

		// Verify the message was updated by searching for it
		searchURL := fmt.Sprintf("/localization/v1/messages?tenantId=%s&module=%s&locale=%s&codes=test-code-1", tenantID, module, locale)
		req, err = http.NewRequest("GET", searchURL, nil)
		require.NoError(t, err)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var searchResponse dtos.SearchMessagesResponse
		err = json.Unmarshal(w.Body.Bytes(), &searchResponse)
		require.NoError(t, err)
		assert.Len(t, searchResponse.Messages, 1)
		assert.Equal(t, "Updated Test Message 1", searchResponse.Messages[0].Message)
	})

	// Step 7: Delete messages
	t.Run("Delete messages", func(t *testing.T) {
		// Create delete messages
		deleteMessages := []dtos.DeleteMessage{
			{
				Code:   "test-code-2",
				Module: module,
				Locale: locale,
			},
		}

		// Create request body
		reqBody := dtos.DeleteMessagesRequest{
			RequestInfo: requestInfo,
			TenantId:    tenantID,
			Messages:    deleteMessages,
		}
		jsonBody, err := json.Marshal(reqBody)
		require.NoError(t, err)

		// Create request
		req, err := http.NewRequest("POST", "/localization/messages/v1/_delete", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response
		var response dtos.DeleteMessagesResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)

		// Verify the message was deleted by searching for all messages
		searchURL := fmt.Sprintf("/localization/v1/messages?tenantId=%s&module=%s&locale=%s", tenantID, module, locale)
		req, err = http.NewRequest("GET", searchURL, nil)
		require.NoError(t, err)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var searchResponse dtos.SearchMessagesResponse
		err = json.Unmarshal(w.Body.Bytes(), &searchResponse)
		require.NoError(t, err)

		// We should have 3 messages left (1 updated + 2 created new ones)
		assert.Len(t, searchResponse.Messages, 3)

		// test-code-2 should not be in the results
		for _, msg := range searchResponse.Messages {
			assert.NotEqual(t, "test-code-2", msg.Code)
		}
	})

	// Step 8: Bust cache
	t.Run("Bust cache", func(t *testing.T) {
		// Create request
		req, err := http.NewRequest("POST", "/localization/messages/cache-bust", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response
		var response dtos.CacheBustResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response.Success)
	})
}

// setupInMemoryDB creates an in-memory SQLite database for testing
func setupInMemoryDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	// Create the messages table
	_, err = db.Exec(`
		CREATE TABLE message (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			tenant_id TEXT NOT NULL,
			module TEXT NOT NULL,
			locale TEXT NOT NULL,
			code TEXT NOT NULL,
			message TEXT NOT NULL,
			created_by INTEGER,
			created_date TIMESTAMP NOT NULL,
			last_modified_by INTEGER,
			last_modified_date TIMESTAMP NOT NULL,
			UNIQUE(tenant_id, locale, module, code)
		);
		CREATE INDEX idx_message_tenant_module_locale ON message(tenant_id, module, locale);
		CREATE INDEX idx_message_tenant_locale ON message(tenant_id, locale);
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}
