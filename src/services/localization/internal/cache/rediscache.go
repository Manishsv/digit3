package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"localization/internal/core/domain"
	"localization/internal/core/ports"
)

type RedisMessageCache struct {
	client *redis.Client
}

func NewRedisMessageCache(client *redis.Client) ports.MessageCache {
	return &RedisMessageCache{
		client: client,
	}
}

func (c *RedisMessageCache) GetMessages(ctx context.Context, tenantID, module, locale string) ([]domain.Message, error) {
	key := buildCacheKey(tenantID, module, locale)
	start := time.Now()

	val, err := c.client.Get(ctx, key).Result()
	duration := time.Since(start)

	if err == redis.Nil {
		log.Printf("REDIS: MISS - key:%s (duration: %v)", key, duration)
		return nil, ports.ErrCacheMiss
	} else if err != nil {
		log.Printf("REDIS: ERROR - key:%s (error: %v, duration: %v)", key, err, duration)
		return nil, err
	}

	var messages []domain.Message
	err = json.Unmarshal([]byte(val), &messages)
	if err != nil {
		log.Printf("REDIS: UNMARSHAL ERROR - key:%s (error: %v)", key, err)
		return nil, err
	}

	log.Printf("REDIS: HIT - key:%s (found %d messages, duration: %v)", key, len(messages), duration)
	return messages, nil
}

func (c *RedisMessageCache) SetMessages(ctx context.Context, tenantID, module, locale string, messages []domain.Message) error {
	key := buildCacheKey(tenantID, module, locale)
	start := time.Now()

	val, err := json.Marshal(messages)
	if err != nil {
		log.Printf("REDIS: MARSHAL ERROR - key:%s (error: %v)", key, err)
		return err
	}

	err = c.client.Set(ctx, key, val, 24*time.Hour).Err()
	duration := time.Since(start)

	if err != nil {
		log.Printf("REDIS: SET ERROR - key:%s (error: %v, duration: %v)", key, err, duration)
		return err
	}

	log.Printf("REDIS: SET - key:%s (stored %d messages, duration: %v)", key, len(messages), duration)
	return nil
}

func (c *RedisMessageCache) Invalidate(ctx context.Context, tenantID, module, locale string) error {
	key := buildCacheKey(tenantID, module, locale)
	start := time.Now()

	err := c.client.Del(ctx, key).Err()
	duration := time.Since(start)

	if err != nil {
		log.Printf("REDIS: INVALIDATE ERROR - key:%s (error: %v, duration: %v)", key, err, duration)
		return err
	}

	log.Printf("REDIS: INVALIDATE - key:%s (duration: %v)", key, duration)
	return nil
}

func (c *RedisMessageCache) BustCache(ctx context.Context, tenantID, module, locale string) error {
	// Busting cache for a specific module and locale combination
	if module != "" && locale != "" {
		key := buildCacheKey(tenantID, module, locale)
		return c.client.Del(ctx, key).Err()
	}

	// Busting cache for an entire tenant (potentially slow, use with caution)
	pattern := tenantID + ":*:*"
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	return nil
}
