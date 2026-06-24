package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"url-shortener/internal/model"
)

var ErrCacheMiss = errors.New("cache miss")

const (
	keyPrefixURL      = "url:"       // url:{shortCode}     → model.URL JSON
	keyPrefixNotFound = "notfound:"  // notfound:{shortCode} → "1" (negative cache)
	notFoundTTL       = 5 * time.Minute
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(addr, password string, db int, ttlSeconds int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to Redis: %w", err)
	}

	log.Info().Str("addr", addr).Msg("Redis connected successfully")

	return &RedisCache{
		client: client,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}, nil
}

// ── URL caching ───────────────────────────────────────────────────

func (c *RedisCache) GetURL(ctx context.Context, shortCode string) (*model.URL, error) {
	key := keyPrefixURL + shortCode

	val, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("redis GET: %w", err)
	}

	var u model.URL
	if err := json.Unmarshal([]byte(val), &u); err != nil {
		return nil, fmt.Errorf("unmarshaling URL: %w", err)
	}

	return &u, nil
}

func (c *RedisCache) SetURL(ctx context.Context, u *model.URL) error {
	key := keyPrefixURL + u.ShortCode

	data, err := json.Marshal(u)
	if err != nil {
		return fmt.Errorf("marshaling URL: %w", err)
	}

	// If URL has expiry, use that as TTL — don't cache past expiry
	ttl := c.ttl
	if u.ExpiresAt != nil {
		remaining := time.Until(*u.ExpiresAt)
		if remaining <= 0 {
			return nil // already expired, don't cache
		}
		if remaining < ttl {
			ttl = remaining
		}
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisCache) DeleteURL(ctx context.Context, shortCode string) error {
	keys := []string{
		keyPrefixURL + shortCode,
		keyPrefixNotFound + shortCode,
	}
	return c.client.Del(ctx, keys...).Err()
}

// ── Negative caching ──────────────────────────────────────────────
// Cache "not found" results to avoid hammering DB with bad codes

func (c *RedisCache) SetNotFound(ctx context.Context, shortCode string) error {
	key := keyPrefixNotFound + shortCode
	return c.client.Set(ctx, key, "1", notFoundTTL).Err()
}

func (c *RedisCache) IsNotFound(ctx context.Context, shortCode string) bool {
	key := keyPrefixNotFound + shortCode
	val, err := c.client.Get(ctx, key).Result()
	return err == nil && val == "1"
}

// ── Cache stats (for /health endpoint) ───────────────────────────

func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}