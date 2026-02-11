package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"localization/internal/core/domain"
)

// MockMessageRepository is a mock implementation of the MessageRepository interface
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) SaveMessages(ctx context.Context, messages []domain.Message) error {
	args := m.Called(ctx, messages)
	return args.Error(0)
}

func (m *MockMessageRepository) FindMessages(ctx context.Context, tenantID, module, locale string) ([]domain.Message, error) {
	args := m.Called(ctx, tenantID, module, locale)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Message), args.Error(1)
}

func (m *MockMessageRepository) FindMessagesByCode(ctx context.Context, tenantID, locale string, codes []string) ([]domain.Message, error) {
	args := m.Called(ctx, tenantID, locale, codes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Message), args.Error(1)
}

// MockMessageCache is a mock implementation of the MessageCache interface
type MockMessageCache struct {
	mock.Mock
}

func (m *MockMessageCache) GetMessages(ctx context.Context, tenantID, module, locale string) ([]domain.Message, error) {
	args := m.Called(ctx, tenantID, module, locale)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Message), args.Error(1)
}

func (m *MockMessageCache) SetMessages(ctx context.Context, messages []domain.Message) error {
	args := m.Called(ctx, messages)
	return args.Error(0)
}

func (m *MockMessageCache) Invalidate(ctx context.Context, tenantID, module, locale string) error {
	args := m.Called(ctx, tenantID, module, locale)
	return args.Error(0)
}

// Test data for all tests
var (
	testTenantID = "DEFAULT"
	testModule   = "test-module"
	testLocale   = "en_IN"
	testMessages = []domain.Message{
		{
			ID:               1,
			TenantID:         testTenantID,
			Module:           testModule,
			Locale:           testLocale,
			Code:             "test-code-1",
			Message:          "Test Message 1",
			CreatedBy:        1,
			CreatedDate:      time.Now(),
			LastModifiedBy:   1,
			LastModifiedDate: time.Now(),
		},
		{
			ID:               2,
			TenantID:         testTenantID,
			Module:           testModule,
			Locale:           testLocale,
			Code:             "test-code-2",
			Message:          "Test Message 2",
			CreatedBy:        1,
			CreatedDate:      time.Now(),
			LastModifiedBy:   1,
			LastModifiedDate: time.Now(),
		},
	}

	testCodes = []string{"test-code-1", "test-code-2"}
)

// TestUpsertMessages tests the UpsertMessages method
func TestUpsertMessages(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockMessageRepository)
		mockCache := new(MockMessageCache)
		service := NewMessageService(mockRepo, mockCache)

		mockRepo.On("SaveMessages", mock.Anything, testMessages).Return(nil).Once()
		mockCache.On("Invalidate", mock.Anything, testTenantID, testModule, testLocale).Return(nil).Once()

		ctx := context.Background()
		result, err := service.UpsertMessages(ctx, testTenantID, testMessages)

		assert.NoError(t, err)
		assert.Equal(t, testMessages, result)
		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})

	t.Run("Repository error", func(t *testing.T) {
		mockRepo := new(MockMessageRepository)
		mockCache := new(MockMessageCache)
		service := NewMessageService(mockRepo, mockCache)

		mockRepo.On("SaveMessages", mock.Anything, testMessages).Return(errors.New("db error")).Once()

		ctx := context.Background()
		result, err := service.UpsertMessages(ctx, testTenantID, testMessages)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
		mockCache.AssertNotCalled(t, "Invalidate")
	})

	t.Run("Cache error (should still succeed)", func(t *testing.T) {
		mockRepo := new(MockMessageRepository)
		mockCache := new(MockMessageCache)
		service := NewMessageService(mockRepo, mockCache)

		mockRepo.On("SaveMessages", mock.Anything, testMessages).Return(nil).Once()
		mockCache.On("Invalidate", mock.Anything, testTenantID, testModule, testLocale).Return(errors.New("cache error")).Once()

		ctx := context.Background()
		result, err := service.UpsertMessages(ctx, testTenantID, testMessages)

		assert.NoError(t, err) // Overall operation should still succeed
		assert.Equal(t, testMessages, result)
		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}

// TestSearchMessages tests the SearchMessages method
func TestSearchMessages(t *testing.T) {
	t.Run("Success with cache hit", func(t *testing.T) {
		mockRepo := new(MockMessageRepository)
		mockCache := new(MockMessageCache)
		service := NewMessageService(mockRepo, mockCache)

		mockCache.On("GetMessages", mock.Anything, testTenantID, testModule, testLocale).Return(testMessages, nil).Once()

		ctx := context.Background()
		result, err := service.SearchMessages(ctx, testTenantID, testModule, testLocale)

		assert.NoError(t, err)
		assert.Equal(t, testMessages, result)
		mockCache.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "FindMessages")
	})

	t.Run("Success with cache miss", func(t *testing.T) {
		mockRepo := new(MockMessageRepository)
		mockCache := new(MockMessageCache)
		service := NewMessageService(mockRepo, mockCache)

		mockCache.On("GetMessages", mock.Anything, testTenantID, testModule, testLocale).Return(nil, errors.New("cache miss")).Once()
		mockRepo.On("FindMessages", mock.Anything, testTenantID, testModule, testLocale).Return(testMessages, nil).Once()
		mockCache.On("SetMessages", mock.Anything, testMessages).Return(nil).Once()

		ctx := context.Background()
		result, err := service.SearchMessages(ctx, testTenantID, testModule, testLocale)

		assert.NoError(t, err)
		assert.Equal(t, testMessages, result)
		mockCache.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Cache error fallback to repository", func(t *testing.T) {
		mockRepo := new(MockMessageRepository)
		mockCache := new(MockMessageCache)
		service := NewMessageService(mockRepo, mockCache)

		mockCache.On("GetMessages", mock.Anything, testTenantID, testModule, testLocale).Return(nil, errors.New("cache error")).Once()
		mockRepo.On("FindMessages", mock.Anything, testTenantID, testModule, testLocale).Return(testMessages, nil).Once()
		mockCache.On("SetMessages", mock.Anything, testMessages).Return(nil).Once()

		ctx := context.Background()
		result, err := service.SearchMessages(ctx, testTenantID, testModule, testLocale)

		assert.NoError(t, err)
		assert.Equal(t, testMessages, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository error", func(t *testing.T) {
		mockRepo := new(MockMessageRepository)
		mockCache := new(MockMessageCache)
		service := NewMessageService(mockRepo, mockCache)

		mockCache.On("GetMessages", mock.Anything, testTenantID, testModule, testLocale).Return(nil, errors.New("cache miss")).Once()
		mockRepo.On("FindMessages", mock.Anything, testTenantID, testModule, testLocale).Return(nil, errors.New("db error")).Once()

		ctx := context.Background()
		result, err := service.SearchMessages(ctx, testTenantID, testModule, testLocale)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
		mockCache.AssertNotCalled(t, "SetMessages")
	})

	t.Run("Empty module bypasses cache", func(t *testing.T) {
		mockRepo := new(MockMessageRepository)
		mockCache := new(MockMessageCache)
		service := NewMessageService(mockRepo, mockCache)

		// Should go directly to repository if module is empty
		mockRepo.On("FindMessages", mock.Anything, testTenantID, "", testLocale).Return(testMessages, nil).Once()

		ctx := context.Background()
		result, err := service.SearchMessages(ctx, testTenantID, "", testLocale)

		assert.NoError(t, err)
		assert.Equal(t, testMessages, result)
		mockRepo.AssertExpectations(t)
		mockCache.AssertNotCalled(t, "GetMessages")
		mockCache.AssertNotCalled(t, "SetMessages")
	})
}

// TestSearchMessagesByCodes tests the SearchMessagesByCodes method
func TestSearchMessagesByCodes(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockMessageRepository)
		mockCache := new(MockMessageCache)
		service := NewMessageService(mockRepo, mockCache)

		codes := []string{"test-code-1", "test-code-2"}
		mockRepo.On("FindMessagesByCode", mock.Anything, testTenantID, testLocale, codes).Return(testMessages, nil).Once()

		ctx := context.Background()
		result, err := service.SearchMessagesByCodes(ctx, testTenantID, testLocale, codes)

		assert.NoError(t, err)
		assert.Equal(t, testMessages, result)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Repository error", func(t *testing.T) {
		mockRepo := new(MockMessageRepository)
		mockCache := new(MockMessageCache)
		service := NewMessageService(mockRepo, mockCache)

		codes := []string{"test-code-1", "test-code-2"}
		mockRepo.On("FindMessagesByCode", mock.Anything, testTenantID, testLocale, codes).Return(nil, errors.New("repository error")).Once()

		ctx := context.Background()
		result, err := service.SearchMessagesByCodes(ctx, testTenantID, testLocale, codes)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
		mockCache.AssertExpectations(t)
	})
}
