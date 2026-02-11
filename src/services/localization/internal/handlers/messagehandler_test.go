package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"localization/internal/common/models"
	"localization/internal/core/domain"
	"localization/pkg/dtos"
)

// MockMessageService is a mock implementation of the MessageService interface
type MockMessageService struct {
	mock.Mock
}

func (m *MockMessageService) UpsertMessages(ctx context.Context, tenantID string, messages []domain.Message) ([]domain.Message, error) {
	args := m.Called(ctx, tenantID, messages)
	return args.Get(0).([]domain.Message), args.Error(1)
}

func (m *MockMessageService) SearchMessages(ctx context.Context, tenantID, module, locale string) ([]domain.Message, error) {
	args := m.Called(ctx, tenantID, module, locale)
	return args.Get(0).([]domain.Message), args.Error(1)
}

func (m *MockMessageService) SearchMessagesByCodes(ctx context.Context, tenantID, locale string, codes []string) ([]domain.Message, error) {
	args := m.Called(ctx, tenantID, locale, codes)
	return args.Get(0).([]domain.Message), args.Error(1)
}

// Test data
var (
	testTenantID = "DEFAULT"
	testModule   = "test-module"
	testLocale   = "en_IN"
	testNow      = time.Now()
	testMessages = []domain.Message{
		{
			ID:               1,
			TenantID:         testTenantID,
			Module:           testModule,
			Locale:           testLocale,
			Code:             "test-code-1",
			Message:          "Test Message 1",
			CreatedBy:        1,
			CreatedDate:      testNow,
			LastModifiedBy:   1,
			LastModifiedDate: testNow,
		},
	}
	testRequestInfo = models.RequestInfo{
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
)

func setupRouter() (*gin.Engine, *MockMessageService) {
	// Create a mock service
	mockService := new(MockMessageService)

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a router with the mock handler
	router := gin.Default()
	handler := NewMessageHandler(mockService)
	apiGroup := router.Group("")
	handler.RegisterRoutes(apiGroup)

	return router, mockService
}

func TestUpsertMessages(t *testing.T) {
	// Setup
	router, mockService := setupRouter()

	t.Run("Success", func(t *testing.T) {
		// Configure mock
		mockService.On("UpsertMessages", mock.Anything, testTenantID, mock.Anything).Return(testMessages, nil).Once()

		// Create request body
		reqBody := dtos.UpsertMessagesRequest{
			RequestInfo: testRequestInfo,
			TenantId:    testTenantID,
			Messages:    testMessages,
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
		assert.Len(t, response.Messages, 1)
		assert.Equal(t, testMessages[0].Code, response.Messages[0].Code)
		assert.Equal(t, testMessages[0].Message, response.Messages[0].Message)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Missing tenant ID", func(t *testing.T) {
		// Create request body without tenant ID
		reqBody := dtos.UpsertMessagesRequest{
			RequestInfo: testRequestInfo,
			Messages:    testMessages,
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

		// Check status code - should be bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Empty messages", func(t *testing.T) {
		// Create request body with empty messages
		reqBody := dtos.UpsertMessagesRequest{
			RequestInfo: testRequestInfo,
			TenantId:    testTenantID,
			Messages:    []domain.Message{},
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

		// Check status code - should be bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Duplicate messages", func(t *testing.T) {
		// Create request body with duplicate messages
		duplicate := testMessages[0]
		messagesWithDuplicate := []domain.Message{duplicate, duplicate}

		reqBody := dtos.UpsertMessagesRequest{
			RequestInfo: testRequestInfo,
			TenantId:    testTenantID,
			Messages:    messagesWithDuplicate,
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

		// Check status code - should be bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		// Configure mock to return error
		mockService.On("UpsertMessages", mock.Anything, testTenantID, mock.Anything).Return([]domain.Message{}, errors.New("service error")).Once()

		// Create request body
		reqBody := dtos.UpsertMessagesRequest{
			RequestInfo: testRequestInfo,
			TenantId:    testTenantID,
			Messages:    testMessages,
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

		// Check status code - should be internal server error
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})
}

func TestSearchMessages(t *testing.T) {
	// Setup
	router, mockService := setupRouter()

	t.Run("Success with JSON body", func(t *testing.T) {
		// Configure mock
		mockService.On("SearchMessages", mock.Anything, testTenantID, testModule, testLocale).Return(testMessages, nil).Once()

		// Create request body
		reqBody := dtos.SearchMessagesRequest{
			RequestInfo: testRequestInfo,
			TenantId:    testTenantID,
			Module:      testModule,
			Locale:      testLocale,
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
		assert.Equal(t, testMessages[0].Code, response.Messages[0].Code)
		assert.Equal(t, testMessages[0].Message, response.Messages[0].Message)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Success with query parameters", func(t *testing.T) {
		// Configure mock
		mockService.On("SearchMessages", mock.Anything, testTenantID, testModule, testLocale).Return(testMessages, nil).Once()

		// Create request with query parameters
		req, err := http.NewRequest("POST", "/localization/messages/v1/_search?tenantId="+testTenantID+"&module="+testModule+"&locale="+testLocale, nil)
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

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Success with codes", func(t *testing.T) {
		// Configure mock
		codes := []string{"test-code-1"}
		mockService.On("SearchMessagesByCodes", mock.Anything, testTenantID, testLocale, codes).Return(testMessages, nil).Once()

		// Create request body with codes
		reqBody := dtos.SearchMessagesRequest{
			RequestInfo: testRequestInfo,
			TenantId:    testTenantID,
			Locale:      testLocale,
			Codes:       codes,
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

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Missing tenant ID", func(t *testing.T) {
		// Create request body without tenant ID
		reqBody := dtos.SearchMessagesRequest{
			RequestInfo: testRequestInfo,
			Module:      testModule,
			Locale:      testLocale,
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

		// Check status code - should be bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Service error", func(t *testing.T) {
		// Configure mock to return error
		mockService.On("SearchMessages", mock.Anything, testTenantID, testModule, testLocale).Return([]domain.Message{}, errors.New("service error")).Once()

		// Create request body
		reqBody := dtos.SearchMessagesRequest{
			RequestInfo: testRequestInfo,
			TenantId:    testTenantID,
			Module:      testModule,
			Locale:      testLocale,
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

		// Check status code - should be internal server error
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})
}
