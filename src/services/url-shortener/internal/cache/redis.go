package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(addr, password string, db int, ttl time.Duration) Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &redisCache{
		client: rdb,
		ttl:    ttl,
	}
}

func (r *redisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", errors.New("key not found")
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

func (r *redisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = r.ttl
	}
	return r.client.Set(ctx, key, value, ttl).Err()
}
