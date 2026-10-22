// Package store keeps short links and rate-limit counters in Redis.
package store

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound is returned when a short code does not exist or has expired.
var ErrNotFound = errors.New("not found")

const (
	linkPrefix  = "link:"
	quotaPrefix = "quota:"
)

// Store is the storage the HTTP API depends on.
type Store interface {
	// SaveLink stores url under code. It returns false if the code is already taken.
	SaveLink(ctx context.Context, code, url string, ttl time.Duration) (bool, error)
	// GetLink returns the original URL for code.
	GetLink(ctx context.Context, code string) (string, error)
	// Take spends one request from the client's quota for the current window.
	// It returns how many requests are left and when the window resets.
	Take(ctx context.Context, client string, quota int, window time.Duration) (left int, reset time.Duration, err error)
	Ping(ctx context.Context) error
}

// Redis implements Store on top of a Redis client.
type Redis struct {
	rdb *redis.Client
}

func NewRedis(rdb *redis.Client) *Redis {
	return &Redis{rdb: rdb}
}

func (r *Redis) SaveLink(ctx context.Context, code, url string, ttl time.Duration) (bool, error) {
	return r.rdb.SetNX(ctx, linkPrefix+code, url, ttl).Result()
}

func (r *Redis) GetLink(ctx context.Context, code string) (string, error) {
	url, err := r.rdb.Get(ctx, linkPrefix+code).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrNotFound
	}
	return url, err
}

func (r *Redis) Take(ctx context.Context, client string, quota int, window time.Duration) (int, time.Duration, error) {
	key := quotaPrefix + client
	// INCR and EXPIRE NX in one round trip: the window starts with the first request.
	pipe := r.rdb.TxPipeline()
	used := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, window)
	ttl := pipe.TTL(ctx, key)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, 0, err
	}
	return quota - int(used.Val()), ttl.Val(), nil
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.rdb.Ping(ctx).Err()
}
