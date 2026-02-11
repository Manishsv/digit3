package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"localization/internal/core/domain"
	"localization/internal/core/ports"
)

// Redis key expiration time
const keyExpiration = 24 * time.Hour

// RedisCacheImpl implements MessageCache using Redis
type RedisCacheImpl struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(client *redis.Client) ports.MessageCache {
	return &RedisCacheImpl{
		client: client,
	}
}

// buildKey creates a cache key from tenant, module, and locale
func buildKey(tenantID, module, locale string) string {
	return fmt.Sprintf("messages:%s:%s:%s", tenantID, module, locale)
}

// SetMessages adds messages to the cache
func (c *RedisCacheImpl) SetMessages(ctx context.Context, tenantID, module, locale string, messages []domain.Message) error {
	if len(messages) == 0 {
		return nil
	}

	key := buildKey(tenantID, module, locale)

	// Serialize messages to JSON
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	// Store in Redis with expiration
	return c.client.Set(ctx, key, messagesJSON, keyExpiration).Err()
}

// GetMessages retrieves messages from the cache
func (c *RedisCacheImpl) GetMessages(ctx context.Context, tenantID, module, locale string) ([]domain.Message, error) {
	key := buildKey(tenantID, module, locale)

	// Get data from Redis
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Key does not exist, return empty slice
			return []domain.Message{}, nil
		}
		return nil, err
	}

	// Deserialize JSON to messages
	var messages []domain.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// Invalidate removes cached messages for a specific tenant+module+locale
func (c *RedisCacheImpl) Invalidate(ctx context.Context, tenantID, module, locale string) error {
	key := buildKey(tenantID, module, locale)
	return c.client.Del(ctx, key).Err()
}

// BustCache clears the cache based on the provided tenant, module, and locale.
// Module and locale are optional. If not provided, all entries for the tenant will be cleared.
func (c *RedisCacheImpl) BustCache(ctx context.Context, tenantID, module, locale string) error {
	if tenantID == "" {
		return fmt.Errorf("tenantID is required to bust cache")
	}

	modulePattern := module
	if modulePattern == "" {
		modulePattern = "*"
	}

	localePattern := locale
	if localePattern == "" {
		localePattern = "*"
	}

	pattern := buildKey(tenantID, modulePattern, localePattern)

	var cursor uint64
	var keys []string
	for {
		var batch []string
		var err error

		// Scan for keys matching the pattern
		batch, cursor, err = c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	// Delete the keys if any are found
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}

	return nil
}
