package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

type inMemoryCache struct {
	mu    sync.RWMutex
	items map[string]string
}

func NewInMemoryCache() Cache {
	return &inMemoryCache{
		items: make(map[string]string),
	}
}

func (m *inMemoryCache) Get(ctx context.Context, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return v, nil
}

func (m *inMemoryCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = value
	return nil
}
