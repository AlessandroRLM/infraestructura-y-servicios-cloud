package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pinger is the readiness seam for dependency health checks.
// Both *pgxpool.Pool and the Redis adapter implement this interface.
type Pinger interface {
	Ping(ctx context.Context) error
}

// NewPool constructs a pgxpool.Pool with an explicitly bounded MaxConns.
// Uses pgx default query exec mode (prepared statements are safe without a pooler).
func NewPool(ctx context.Context, dsn string, maxConns int) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = int32(maxConns) //nolint:gosec // MaxConns is validated to be > 0 by config.Load.
	return pgxpool.NewWithConfig(ctx, cfg)
}
