package storage

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

type RedisBackend struct {
	client *redis.Client
}

func NewRedisBackend(client *redis.Client) *RedisBackend {
	return &RedisBackend{client: client}
}

func (r *RedisBackend) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (r *RedisBackend) Set(ctx context.Context, key string, value []byte) error {
	// TTL is ignored (0 = infinite)
	return r.client.Set(ctx, key, value, 0).Err()
}

func (r *RedisBackend) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}