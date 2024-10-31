package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(ip string, port int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:                  fmt.Sprintf("%s:%d", ip, port),
		ContextTimeoutEnabled: true,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	redisClient := &RedisClient{
		client: client,
	}
	return redisClient, nil
}

func (rs *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := rs.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return "", nil
	}
	return val, nil
}

func (rs *RedisClient) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	return rs.client.Set(ctx, key, val, ttl).Err()
}

func (rs *RedisClient) Close() {
	rs.client.Close()
}
