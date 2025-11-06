package db

import (
	"context"
	"fxrates/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreatePool(ctx context.Context, cfg config.DbServer) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, cfg.GetFormattedParams())
	if err != nil {
		panic(err)
	} // todo: ...
	return pool
}
