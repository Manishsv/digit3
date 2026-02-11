package cache

import (
    "context"
    "github.com/go-redis/redis/v8"
    "errors"
    "fmt"
    "log"
)

type RedisCache struct {
    client *redis.Client
}

func NewRedisCache(addr, password string, db int) *RedisCache {
    rdb := redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: password,
        DB:       db,
    })
    
    // Test connection
    ctx := context.Background()
    if err := rdb.Ping(ctx).Err(); err != nil {
        log.Printf("Redis connection failed: %v", err)
    } else {
        log.Printf("Redis connected successfully to %s", addr)
    }
    
    return &RedisCache{client: rdb}
}

func (c *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
    val, err := c.client.Get(ctx, key).Result()
    if err == redis.Nil {
        return nil, errors.New("key not found")
    } else if err != nil {
        return nil, fmt.Errorf("redis get error: %w", err)
    }
    return val, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}) error {
    strVal, ok := value.(string)
    if !ok {
        return errors.New("redis cache only supports string values")
    }
    log.Printf("Redis Set: key=%s", key)
    return c.client.Set(ctx, key, strVal, 0).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
    return c.client.Del(ctx, key).Err()
}

// DeletePattern deletes all keys matching the given pattern
// Note: KEYS command can be expensive in production, consider using SCAN for large datasets
func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
    log.Printf("Redis DeletePattern called with pattern: %s", pattern)
    
    // Use SCAN instead of KEYS for better performance in production
    var cursor uint64
    var keys []string
    
    for {
        scanKeys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
        if err != nil {
            log.Printf("Redis scan error: %v", err)
            return fmt.Errorf("redis scan error: %w", err)
        }
        
        keys = append(keys, scanKeys...)
        cursor = nextCursor
        
        if cursor == 0 {
            break
        }
    }
    
    log.Printf("Redis DeletePattern found %d keys to delete: %v", len(keys), keys)
    
    if len(keys) > 0 {
        err := c.client.Del(ctx, keys...).Err()
        if err != nil {
            log.Printf("Redis delete error: %v", err)
        } else {
            log.Printf("Redis successfully deleted %d keys", len(keys))
        }
        return err
    }
    log.Printf("Redis DeletePattern: no keys found matching pattern %s", pattern)
    return nil
}

// DeleteByPrefix deletes all keys with the given prefix
func (c *RedisCache) DeleteByPrefix(ctx context.Context, prefix string) error {
    log.Printf("Redis DeleteByPrefix called with prefix: %s", prefix)
    
    // Use SCAN to find keys with prefix
    pattern := prefix + "*"
    var cursor uint64
    var keys []string
    
    for {
        scanKeys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
        if err != nil {
            log.Printf("Redis scan error: %v", err)
            return fmt.Errorf("redis scan error: %w", err)
        }
        
        keys = append(keys, scanKeys...)
        cursor = nextCursor
        
        if cursor == 0 {
            break
        }
    }
    
    log.Printf("Redis DeleteByPrefix found %d keys to delete: %v", len(keys), keys)
    
    if len(keys) > 0 {
        err := c.client.Del(ctx, keys...).Err()
        if err != nil {
            log.Printf("Redis delete error: %v", err)
        } else {
            log.Printf("Redis successfully deleted %d keys by prefix", len(keys))
        }
        return err
    }
    log.Printf("Redis DeleteByPrefix: no keys found with prefix %s", prefix)
    return nil
} 