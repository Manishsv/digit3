package cache

import (
    "context"
    "errors"
    "strings"
    "sync"
)

type InMemoryCache struct {
    data map[string]interface{}
    mu   sync.RWMutex
}

func NewInMemoryCache() *InMemoryCache {
    return &InMemoryCache{
        data: make(map[string]interface{}),
    }
}

func (c *InMemoryCache) Get(ctx context.Context, key string) (interface{}, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    value, exists := c.data[key]
    if !exists {
        return nil, errors.New("key not found")
    }
    return value, nil
}

func (c *InMemoryCache) Set(ctx context.Context, key string, value interface{}) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.data[key] = value
    return nil
}

func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    delete(c.data, key)
    return nil
}

// DeleteByPrefix deletes all keys with the given prefix
func (c *InMemoryCache) DeleteByPrefix(ctx context.Context, prefix string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    keysToDelete := make([]string, 0)
    for key := range c.data {
        if strings.HasPrefix(key, prefix) {
            keysToDelete = append(keysToDelete, key)
        }
    }
    
    for _, key := range keysToDelete {
        delete(c.data, key)
    }
    return nil
}

// DeletePattern deletes all keys matching the pattern (simple wildcard support)
func (c *InMemoryCache) DeletePattern(ctx context.Context, pattern string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    keysToDelete := make([]string, 0)
    for key := range c.data {
        if matchPattern(key, pattern) {
            keysToDelete = append(keysToDelete, key)
        }
    }
    
    for _, key := range keysToDelete {
        delete(c.data, key)
    }
    return nil
}

// matchPattern provides simple wildcard matching
func matchPattern(text, pattern string) bool {
    if pattern == "*" {
        return true
    }
    
    parts := strings.Split(pattern, "*")
    if len(parts) == 1 {
        return text == pattern
    }
    
    // Check if text starts with first part and ends with last part
    if !strings.HasPrefix(text, parts[0]) {
        return false
    }
    if !strings.HasSuffix(text, parts[len(parts)-1]) {
        return false
    }
    
    // Check middle parts exist in order
    current := text
    for i := 1; i < len(parts)-1; i++ {
        if parts[i] == "" {
            continue
        }
        idx := strings.Index(current, parts[i])
        if idx == -1 {
            return false
        }
        current = current[idx+len(parts[i]):]
    }
    
    return true
} 