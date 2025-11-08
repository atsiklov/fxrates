package db

import (
	"context"
	"fxrates/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreatePool(ctx context.Context, cfg config.DbServer) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.GetConnectionStr())
	if err != nil {
		return nil, err
	}
	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
