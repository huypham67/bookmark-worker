package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type redisRepository struct {
	client *redis.Client
}

// NewRedis returns a Repository backed by Redis.
func NewRedis(client *redis.Client) Repository {
	return &redisRepository{client: client}
}

func (r *redisRepository) DeleteCacheByHashKey(ctx context.Context, hashKey string) error {
	return r.client.Del(ctx, hashKey).Err()
}
