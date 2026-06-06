// Package redis provides Redis client construction and a Pinger adapter.
package redis

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/AlessandroRLM/infraestructura-y-servicios-cloud/backend/internal/platform/db"
)

// NewClient parses the Redis URL and returns a configured go-redis/v9 client.
func NewClient(url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opts), nil
}

type redisPinger struct {
	client *redis.Client
}

// Ping implements db.Pinger by calling the Redis PING command.
func (r *redisPinger) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// NewPinger parses the Redis URL and returns a db.Pinger backed by a go-redis client.
// This is the preferred constructor for use in server and main.
func NewPinger(url string) (db.Pinger, error) {
	client, err := NewClient(url)
	if err != nil {
		return nil, err
	}
	return &redisPinger{client: client}, nil
}
