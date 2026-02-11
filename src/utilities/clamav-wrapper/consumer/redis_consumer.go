package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"

	"clamav-wrapper/config"
	"clamav-wrapper/models"
)

// RedisConsumer implements the MessageConsumer interface for Redis, using BLPOP on a list.
type RedisConsumer struct {
	client  *redis.Client
	key     string // Redis list key to pop messages from
	handler func(bucketName string, objectKeyEncoded string) error
}

// NewRedisConsumer creates and configures a new RedisConsumer.
// It initializes a Redis client, pings the server, and stores configuration.
func NewRedisConsumer(cfg config.RedisConfig, handlerFunc func(bucketName string, objectKeyEncoded string) error) (*RedisConsumer, error) {
	if handlerFunc == nil {
		return nil, fmt.Errorf("handler function cannot be nil for RedisConsumer")
	}

	opt := &redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	client := redis.NewClient(opt)

	// Ping Redis to check connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5-second timeout for ping
	defer cancel()
	if _, err := client.Ping(ctx).Result(); err != nil {
		client.Close() // Close client if ping fails
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", cfg.Address, err)
	}

	log.Printf("Successfully connected to Redis at %s", cfg.Address)

	return &RedisConsumer{
		client:  client,
		key:     cfg.Key,
		handler: handlerFunc,
	}, nil
}

// StartConsumer begins consuming messages from the configured Redis list using BLPOP.
func (rc *RedisConsumer) StartConsumer() error {
	if rc.handler == nil {
		return fmt.Errorf("RedisConsumer's handler is not set")
	}
	if rc.client == nil {
		return fmt.Errorf("RedisConsumer's client is not initialized")
	}

	log.Printf("Starting Redis consumer for key %s using BLPOP", rc.key)
	ctx := context.Background()

	for {
		results, err := rc.client.BLPop(ctx, 0*time.Second, rc.key).Result()
		if err != nil {
			if err == redis.Nil { // redis.Nil is returned on timeout if BLPop timeout > 0. With 0, it blocks indefinitely.
				// This path might be taken if client is closing / context is cancelled.
				log.Printf("Redis BLPop returned nil (key: %s), client might be closing or context cancelled.", rc.key)
				// Check context error for graceful shutdown
				if ctx.Err() != nil {
					log.Printf("Context error detected: %v. Shutting down Redis consumer.", ctx.Err())
					return nil // Graceful shutdown
				}
				// If no context error, it might be an unexpected nil, possibly retry or log more.
				// For indefinite timeout (0), redis.Nil is not expected unless client is closed.
				return fmt.Errorf("unexpected redis.Nil from BLPop with 0 timeout for key %s: %w", rc.key, err)
			}
			log.Printf("Error receiving message from Redis key %s using BLPop: %v", rc.key, err)
			// Add a small delay before retrying to prevent tight loop on persistent errors.
			time.Sleep(1 * time.Second)
			continue
		}

		if len(results) < 2 {
			log.Printf("Error: BLPop for key %s returned unexpected results length: %d. Expected 2 (key, value).", rc.key, len(results))
			continue
		}
		// results[0] is the key name, results[1] is the value (JSON payload string)
		payload := results[1]

		var event models.RedisEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			log.Printf("Error unmarshalling RedisEvent array from Redis: %v. Payload: %s", err, payload)
			continue
		}

		if len(event) == 0 {
			log.Printf("Received empty notifications array from Redis key %s. Payload: %s", rc.key, payload)
			continue
		}

		for _, redisEvent := range event {
			if len(redisEvent.Event) == 0 {
				log.Printf("Received empty event array from Redis key %s. Payload: %s", rc.key, payload)
				continue
			}
			for _, event := range redisEvent.Event {
				if err := rc.handler(event.S3.Bucket.Name, event.S3.Object.Key); err != nil {
					log.Printf("Error processing event from Redis: %v", err)
					// Continue processing next message
				}
			}
		}
	}
}

// Close gracefully shuts down the Redis consumer by closing the Redis client.
func (rc *RedisConsumer) Close() error {
	if rc.client != nil {
		log.Println("Closing Redis consumer client.")
		return rc.client.Close()
	}
	return nil
}
