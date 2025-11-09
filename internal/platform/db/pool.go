package db

import (
	"context"
	"fxrates/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreatePoolAndPing(ctx context.Context, cfg config.DbServer) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.GetConnectionStr())
	if err != nil {
		return nil, err
	}
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
