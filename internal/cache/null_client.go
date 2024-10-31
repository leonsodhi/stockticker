package cache

import (
	"context"
	"time"
)

type NullClient struct{}

func NewNullClient(ip string, port int) (*NullClient, error) {
	return &NullClient{}, nil
}

func (n *NullClient) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (n *NullClient) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	return nil
}

func (n *NullClient) Close() {}
