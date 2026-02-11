package cache

import "context"

// Cache is a generic cache interface
// All implementations (in-memory, Redis) must satisfy this
// Value is interface{} for flexibility, but can be changed to []byte or string if needed

type Cache interface {
    Get(ctx context.Context, key string) (interface{}, error)
    Set(ctx context.Context, key string, value interface{}) error
    Delete(ctx context.Context, key string) error
}

// ExtendedCache provides additional cache operations
type ExtendedCache interface {
    Cache
    DeletePattern(ctx context.Context, pattern string) error
    DeleteByPrefix(ctx context.Context, prefix string) error
} 