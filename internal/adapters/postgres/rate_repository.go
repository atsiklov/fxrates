package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"fxrates/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RateRepository struct {
	pool *pgxpool.Pool
}

func (r *RateRepository) GetByCodes(ctx context.Context, base string, quote string) (domain.Rate, error) {
	const q = `
        select fp.id, fp.base, fp.quote, round(flr.value, 4) as value, flr.updated_at
        from fx_last_rates flr join fx_pairs fp on flr.pair_id = fp.id
        where fp.base = $1 and fp.quote = $2;
    `

	var rate domain.Rate
	if err := r.pool.QueryRow(ctx, q, base, quote).Scan(
		&rate.PairID,
		&rate.Base,
		&rate.Quote,
		&rate.Value,
		&rate.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Rate{}, domain.ErrRateNotFound
		}
		return domain.Rate{}, fmt.Errorf("failed to select rate for pair %q/%q: %w", base, quote, err)
	}

	return rate, nil
}

func (r *RateRepository) GetByUpdateID(ctx context.Context, updateID uuid.UUID) (domain.Rate, domain.RateUpdateStatus, error) {
	const q = `
            select fp.id, 
               fp.base, 
               fp.quote, 
               case when fru.status = 'applied' then round(fru.value, 4) end as value,
               fru.updated_at, 
               fru.status
            from fx_rate_updates fru join fx_pairs fp on fru.pair_id = fp.id
            where fru.update_id = $1;
        `

	var rate domain.Rate
	var status domain.RateUpdateStatus
	var value sql.NullFloat64

	if err := r.pool.QueryRow(ctx, q, updateID).Scan(
		&rate.PairID,
		&rate.Base,
		&rate.Quote,
		&value,
		&rate.UpdatedAt,
		&status,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Rate{}, "", domain.ErrRateNotFound
		}
		return domain.Rate{}, "", fmt.Errorf("failed to select rate for update ID %q: %w", updateID, err)
	}
	if value.Valid {
		rate.Value = value.Float64
	} else {
		rate.Value = -1 // explicitly set bad value
	}
	return rate, status, nil
}

func NewRateRepository(pool *pgxpool.Pool) *RateRepository {
	return &RateRepository{pool: pool}
}
