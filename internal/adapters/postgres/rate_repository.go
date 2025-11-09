package postgres

import (
	"context"
	"errors"
	"fmt"
	"fxrates/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RateRepository struct {
	pool *pgxpool.Pool
}

func (r *RateRepository) GetByCodes(ctx context.Context, base string, quote string) (*domain.AppliedRate, error) {
	const q = `
        select fp.id, fp.base, fp.quote, flr.value, flr.updated_at
        from fx_last_rates flr join fx_pairs fp on flr.pair_id = fp.id
        where fp.base = $1 and fp.quote = $2;
    `

	var rate domain.AppliedRate
	if err := r.pool.QueryRow(ctx, q, base, quote).Scan(
		&rate.PairID,
		&rate.Base,
		&rate.Quote,
		&rate.Value,
		&rate.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRateNotFound
		}
		return nil, fmt.Errorf("failed to select rate for pair %q/%q: %w", base, quote, err)
	}

	return &rate, nil
}

func NewRateRepository(pool *pgxpool.Pool) *RateRepository {
	return &RateRepository{pool: pool}
}
