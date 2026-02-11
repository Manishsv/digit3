package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"localization/internal/core/domain"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	// Start a mini Redis server for testing
	miniRedis, err := miniredis.Run()
	require.NoError(t, err)

	// Create a Redis client connected to the mini Redis server
	client := redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})

	return miniRedis, client
}

func TestSetMessages(t *testing.T) {
	// Setup mini redis
	miniRedis, client := setupTestRedis(t)
	defer miniRedis.Close()

	// Create cache with redis client
	cache := NewRedisCache(client)

	// Test data
	tenantID := "DEFAULT"
	module := "test-module"
	locale := "en_IN"
	now := time.Now()
	messages := []domain.Message{
		{
			ID:               1,
			TenantID:         tenantID,
			Module:           module,
			Locale:           locale,
			Code:             "test-code-1",
			Message:          "Test Message 1",
			CreatedBy:        1,
			CreatedDate:      now,
			LastModifiedBy:   1,
			LastModifiedDate: now,
		},
	}

	// Test scenarios
	t.Run("Success", func(t *testing.T) {
		// Call the method under test
		err := cache.SetMessages(context.Background(), tenantID, module, locale, messages)

		// Assertions
		assert.NoError(t, err)

		// Verify data was stored in Redis
		key := buildKey(tenantID, module, locale)
		assert.True(t, miniRedis.Exists(key))
	})

	t.Run("Empty messages", func(t *testing.T) {
		// Call the method under test with empty messages
		err := cache.SetMessages(context.Background(), tenantID, module, locale, []domain.Message{})

		// Assertions - should not error but also not store anything
		assert.NoError(t, err)
	})

	t.Run("Redis error", func(t *testing.T) {
		// Simulate Redis connection error
		miniRedis.Close()

		// Call the method under test
		err := cache.SetMessages(context.Background(), tenantID, module, locale, messages)

		// Assertions
		assert.Error(t, err)
	})
}

func TestGetMessages(t *testing.T) {
	// Setup mini redis
	miniRedis, client := setupTestRedis(t)
	defer miniRedis.Close()

	// Create cache with redis client
	cache := NewRedisCache(client)

	// Test data
	tenantID := "DEFAULT"
	module := "test-module"
	locale := "en_IN"
	now := time.Now()
	messages := []domain.Message{
		{
			ID:               1,
			TenantID:         tenantID,
			Module:           module,
			Locale:           locale,
			Code:             "test-code-1",
			Message:          "Test Message 1",
			CreatedBy:        1,
			CreatedDate:      now,
			LastModifiedBy:   1,
			LastModifiedDate: now,
		},
	}

	t.Run("Cache hit", func(t *testing.T) {
		// First set the messages
		err := cache.SetMessages(context.Background(), tenantID, module, locale, messages)
		require.NoError(t, err)

		// Call the method under test
		retrievedMessages, err := cache.GetMessages(context.Background(), tenantID, module, locale)

		// Assertions
		assert.NoError(t, err)
		assert.Len(t, retrievedMessages, 1)
		assert.Equal(t, messages[0].ID, retrievedMessages[0].ID)
		assert.Equal(t, messages[0].Code, retrievedMessages[0].Code)
		assert.Equal(t, messages[0].Message, retrievedMessages[0].Message)
	})

	t.Run("Cache miss", func(t *testing.T) {
		// Call with non-existent key
		retrievedMessages, err := cache.GetMessages(context.Background(), "nonexistent", module, locale)

		// Assertions - should return empty slice without error
		assert.NoError(t, err)
		assert.Empty(t, retrievedMessages)
	})

	t.Run("Invalid data format", func(t *testing.T) {
		// Set invalid data directly in Redis
		key := buildKey(tenantID, "invalid", locale)
		miniRedis.Set(key, "not valid json")

		// Call the method under test
		retrievedMessages, err := cache.GetMessages(context.Background(), tenantID, "invalid", locale)

		// Assertions
		assert.Error(t, err)
		assert.Empty(t, retrievedMessages)
	})

	t.Run("Redis error", func(t *testing.T) {
		// Simulate Redis connection error
		miniRedis.Close()

		// Call the method under test
		retrievedMessages, err := cache.GetMessages(context.Background(), tenantID, module, locale)

		// Assertions
		assert.Error(t, err)
		assert.Empty(t, retrievedMessages)
	})
}

func TestInvalidate(t *testing.T) {
	// Setup mini redis
	miniRedis, client := setupTestRedis(t)
	defer miniRedis.Close()

	// Create cache with redis client
	cache := NewRedisCache(client)

	// Test data
	tenantID := "DEFAULT"
	module := "test-module"
	locale := "en_IN"
	now := time.Now()
	messages := []domain.Message{
		{
			ID:               1,
			TenantID:         tenantID,
			Module:           module,
			Locale:           locale,
			Code:             "test-code-1",
			Message:          "Test Message 1",
			CreatedBy:        1,
			CreatedDate:      now,
			LastModifiedBy:   1,
			LastModifiedDate: now,
		},
	}

	t.Run("Success", func(t *testing.T) {
		// First set the messages
		err := cache.SetMessages(context.Background(), tenantID, module, locale, messages)
		require.NoError(t, err)

		// Verify data exists
		key := buildKey(tenantID, module, locale)
		assert.True(t, miniRedis.Exists(key))

		// Call the method under test
		err = cache.Invalidate(context.Background(), tenantID, module, locale)

		// Assertions
		assert.NoError(t, err)
		assert.False(t, miniRedis.Exists(key))
	})

	t.Run("Key does not exist", func(t *testing.T) {
		// Call the method under test for non-existent key
		err := cache.Invalidate(context.Background(), "nonexistent", module, locale)

		// Assertions - should not error
		assert.NoError(t, err)
	})

	t.Run("Redis error", func(t *testing.T) {
		// Simulate Redis connection error
		miniRedis.Close()

		// Call the method under test
		err := cache.Invalidate(context.Background(), tenantID, module, locale)

		// Assertions
		assert.Error(t, err)
	})
}

func TestBuildKey(t *testing.T) {
	testCases := []struct {
		name     string
		tenantID string
		module   string
		locale   string
		expected string
	}{
		{
			name:     "Normal key",
			tenantID: "DEFAULT",
			module:   "test-module",
			locale:   "en_IN",
			expected: "messages:DEFAULT:test-module:en_IN",
		},
		{
			name:     "Empty module",
			tenantID: "DEFAULT",
			module:   "",
			locale:   "en_IN",
			expected: "messages:all:DEFAULT:en_IN",
		},
		{
			name:     "Empty tenant",
			tenantID: "",
			module:   "test-module",
			locale:   "en_IN",
			expected: "messages::test-module:en_IN",
		},
		{
			name:     "Empty locale",
			tenantID: "DEFAULT",
			module:   "test-module",
			locale:   "",
			expected: "messages:DEFAULT:test-module:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := buildKey(tc.tenantID, tc.module, tc.locale)
			assert.Equal(t, tc.expected, key)
		})
	}
}
