package cache

import (
	"context"
	"errors"
	"time"
)

type noOpCache struct{}

func NewNoOpCache() Cache {
	return &noOpCache{}
}

func (c *noOpCache) Get(ctx context.Context, key string) (string, error) {
	return "", errors.New("key not found")
}

func (c *noOpCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return nil
}
